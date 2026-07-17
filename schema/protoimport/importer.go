package protoimport

import (
	"github.com/gojekfarm/xtools/errors"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// ImportFiles walks every top-level message in each file descriptor and returns
// one [schema.Def] per message. The proto package becomes the namespace unless
// overridden with [WithNamespace].
func ImportFiles(files []protoreflect.FileDescriptor, opts ...Option) ([]*schema.Def, error) {
	o := buildOptions(opts)

	var defs []*schema.Def
	for _, f := range files {
		msgs := f.Messages()
		for i := range msgs.Len() {
			def, err := importMessage(msgs.Get(i), o)
			if err != nil {
				return nil, err
			}
			defs = append(defs, def)
		}
	}

	return defs, nil
}

// ImportMessage imports a single message descriptor into a [schema.Def].
func ImportMessage(md protoreflect.MessageDescriptor, opts ...Option) (*schema.Def, error) {
	return importMessage(md, buildOptions(opts))
}

func importMessage(md protoreflect.MessageDescriptor, o Options) (*schema.Def, error) {
	ns := o.ns
	if ns == "" {
		ns = string(md.ParentFile().Package())
	}
	if ns == "" {
		return nil, errors.Wrap(ErrNoNamespace,
			"message", string(md.FullName()),
			"fix", "pass WithNamespace to set the target namespace",
		)
	}

	uri, err := core.NewURI(ns, string(md.Name()))
	if err != nil {
		return nil, err
	}

	plan, err := buildMessagePlan(md, []protoreflect.FullName{md.FullName()}, o.allowJSON)
	if err != nil {
		return nil, err
	}

	def := &schema.Def{
		URI:    uri,
		Mode:   schema.ModeStrict,
		Fields: planToFields(plan, ""),
		Annotations: map[string]string{
			"source":        "proto",
			"proto.message": string(md.FullName()),
			"proto.file":    md.ParentFile().Path(),
		},
	}

	if err := def.Validate(); err != nil {
		return nil, err
	}

	return def, nil
}

// planToFields flattens field plans into a schema field map. prefix carries the
// dotted namespace for nested messages; object-array Items form their own
// namespace (built with an empty prefix).
func planToFields(plan []fieldPlan, prefix string) map[string]schema.Field {
	fields := make(map[string]schema.Field)

	for _, fp := range plan {
		name := prefix + fp.attr

		switch fp.kind {
		case kindNested:
			for k, v := range planToFields(fp.items, name+".") {
				fields[k] = v
			}

		case kindObjectArray:
			fields[name] = schema.Field{
				Type:        fp.coreType,
				Annotations: fp.ann,
				Items:       planToFields(fp.items, ""),
			}

		default: // kindLeaf, kindScalarArray
			fields[name] = schema.Field{
				Type:        fp.coreType,
				Annotations: fp.ann,
			}
		}
	}

	return fields
}

// CheckRename compares a re-imported schema against the stored one using proto
// field numbers. A field whose number matches an existing field under a
// different name is a rename; v1 surfaces it as [ErrRename] with instructions
// rather than applying it, because renaming means rewriting stored tuples.
//
// Numbers that are ambiguous (shared by more than one field in a namespace,
// which happens when nested messages flatten) are skipped.
func CheckRename(existing, updated *schema.Def) error {
	byNumber := uniqueNumbers(existing.Fields)

	for name, field := range updated.Fields {
		num, ok := field.Annotations["proto.number"]
		if !ok {
			continue
		}
		oldName, ok := byNumber[num]
		if !ok || oldName == name {
			continue
		}
		if _, stillThere := existing.Fields[name]; stillThere {
			continue // the new name already existed; not a rename
		}
		return errors.Wrap(ErrRename,
			"number", num,
			"from", oldName,
			"to", name,
			"fix", "apply the rename with a tuple-level migration, or add the field under a new number",
		)
	}

	return nil
}

// uniqueNumbers maps proto number → field name, keeping only numbers that
// belong to exactly one field.
func uniqueNumbers(fields map[string]schema.Field) map[string]string {
	names := make(map[string][]string)
	for name, field := range fields {
		if num, ok := field.Annotations["proto.number"]; ok {
			names[num] = append(names[num], name)
		}
	}

	out := make(map[string]string, len(names))
	for num, ns := range names {
		if len(ns) == 1 {
			out[num] = ns[0]
		}
	}
	return out
}
