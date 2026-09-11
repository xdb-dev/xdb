/* The terminal replay on the landing page. demo.json holds sessions that
   `make site-demo` records from real runs of the CLI. Each session is a
   chapter. A chapter shows a still frame under a play button; a click
   types each command and prints the recorded output.

   With prefers-reduced-motion, play shows the full session at once. */
(function () {
  const root = document.getElementById("replay");
  if (!root) return;

  const stage = root.querySelector(".stage");
  const screen = root.querySelector(".screen");
  const overlay = root.querySelector(".overlay");
  const title = overlay.querySelector(".ov-title");
  const meta = overlay.querySelector(".ov-meta");
  const chapters = root.querySelector(".chapters");
  const bar = root.querySelector(".progress span");
  const toggle = root.querySelector("[data-toggle]");
  const transcript = root.querySelector(".transcript");
  const still = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  const NOTE_MS = 500; // after a comment line
  const ENTER_MS = 400; // after a command, before its output
  const LINE_MS = 30; // for each line of output
  const STEP_MS = 1300; // after each step

  // A long command types faster, so that no command takes more than about
  // 2.6 seconds to type.
  const typeMs = (cmd) => Math.min(24, 2600 / cmd.length);

  const cursor = document.createElement("span");
  cursor.className = "cur";

  let episodes = [];
  let index = 0;
  let paused = false;
  let elapsed = 0;
  let total = 1;
  // Each start of a replay gets a new id. A loop with an old id stops.
  let current = 0;

  const wait = (ms) => new Promise((done) => setTimeout(done, ms));
  const lines = (step) => (step.out ? step.out.replace(/\n$/, "").split("\n") : []);

  function durationOf(episode) {
    return episode.steps.reduce((ms, step) => {
      const note = step.note ? NOTE_MS : 0;
      const typing = step.cmd.length * typeMs(step.cmd);
      return ms + note + typing + ENTER_MS + lines(step).length * LINE_MS + STEP_MS;
    }, 0);
  }

  function plainText(episode) {
    return episode.steps
      .map((s) => [s.note ? `# ${s.note}` : "", `$ ${s.cmd}`, ...lines(s)]
        .filter((part) => part !== "")
        .join("\n"))
      .join("\n\n");
  }

  function add(cls, text) {
    const span = document.createElement("span");
    if (cls) span.className = cls;
    span.textContent = text;
    screen.appendChild(span);
    screen.appendChild(cursor);
    screen.scrollTop = screen.scrollHeight;
    return span;
  }

  function renderAll(episode) {
    screen.textContent = "";
    for (const step of episode.steps) {
      if (step.note) add("c", `# ${step.note}\n`);
      add("p", "$ ");
      add("", `${step.cmd}\n`);
      for (const text of lines(step)) add(step.exit ? "e" : "", `${text}\n`);
      add("", "\n");
    }
  }

  function progress(ms) {
    elapsed += ms;
    bar.style.width = `${Math.min(100, (elapsed / total) * 100)}%`;
  }

  // Waits for ms, and longer while the replay is paused. It returns false
  // when a newer replay has started, so the caller stops.
  async function tick(id, ms) {
    await wait(ms);
    while (paused && id === current) await wait(100);
    if (id !== current) return false;
    progress(ms);
    return true;
  }

  function setState(state) {
    stage.dataset.state = state;
    overlay.hidden = state === "playing";
    toggle.hidden = state === "poster" || state === "done";
    toggle.textContent = state === "paused" ? "Play" : "Pause";
  }

  function poster() {
    current++;
    paused = false;
    const episode = episodes[index];

    renderAll(episode);
    screen.scrollTop = 0;
    bar.style.width = "0%";
    title.textContent = episode.title;
    meta.textContent = `${episode.steps.length} commands · ${Math.round(durationOf(episode) / 1000)} s`;
    overlay.setAttribute("aria-label", `Play: ${episode.title}`);
    transcript.textContent = plainText(episode);
    setState("poster");
  }

  function finish() {
    title.textContent = episodes[index].title;
    meta.textContent = "Play again";
    overlay.setAttribute("aria-label", `Play again: ${episodes[index].title}`);
    bar.style.width = "100%";
    setState("done");
  }

  async function play() {
    const id = ++current;
    const episode = episodes[index];
    paused = false;
    elapsed = 0;
    total = durationOf(episode);
    setState("playing");

    if (still) {
      renderAll(episode);
      finish();
      return;
    }

    screen.textContent = "";
    bar.style.width = "0%";

    for (const step of episode.steps) {
      if (step.note) {
        add("c", `# ${step.note}\n`);
        if (!(await tick(id, NOTE_MS))) return;
      }

      add("p", "$ ");
      const cmd = add("", "");
      const ms = typeMs(step.cmd);
      for (const ch of step.cmd) {
        cmd.textContent += ch;
        screen.scrollTop = screen.scrollHeight;
        if (!(await tick(id, ms))) return;
      }
      cmd.textContent += "\n";
      if (!(await tick(id, ENTER_MS))) return;

      for (const text of lines(step)) {
        add(step.exit ? "e" : "", `${text}\n`);
        if (!(await tick(id, LINE_MS))) return;
      }
      add("", "\n");
      if (!(await tick(id, STEP_MS))) return;
    }

    finish();
  }

  function pause() {
    paused = true;
    title.textContent = "Paused";
    meta.textContent = "Click to continue";
    overlay.setAttribute("aria-label", "Continue");
    setState("paused");
  }

  function resume() {
    paused = false;
    setState("playing");
  }

  overlay.addEventListener("click", () => {
    if (stage.dataset.state === "paused") resume();
    else play();
  });

  screen.addEventListener("click", () => {
    if (stage.dataset.state === "playing") pause();
  });

  toggle.addEventListener("click", () => {
    if (stage.dataset.state === "playing") pause();
    else if (stage.dataset.state === "paused") resume();
  });

  function select(i) {
    index = i;
    chapters.querySelectorAll("button").forEach((b, n) => {
      b.setAttribute("aria-selected", String(n === i));
    });
    poster();
  }

  fetch("demo.json")
    .then((res) => res.json())
    .then((data) => {
      episodes = data.episodes;
      episodes.forEach((episode, i) => {
        const button = document.createElement("button");
        button.type = "button";
        button.setAttribute("role", "tab");
        button.textContent = episode.label;
        button.addEventListener("click", () => select(i));
        chapters.appendChild(button);
      });
      select(0);
    })
    .catch(() => {
      overlay.hidden = true;
      screen.textContent = "The replay did not load. The get-started guide shows the same commands.";
    });
})();
