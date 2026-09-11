// Starlight reads this file for the Expressive Code options.
export default {
  themes: ["github-light"],
  styleOverrides: {
    borderRadius: "0",
    borderColor: "var(--line)",
    codeFontFamily: "var(--mono)",
    codeFontSize: "0.8rem",
    codeLineHeight: "1.65",
    uiFontFamily: "var(--mono)",
    codeBackground: "var(--card)",
    frames: {
      shadowColor: "transparent",
      editorTabBarBackground: "var(--bg2)",
      editorTabBarBorderBottomColor: "var(--line)",
      editorActiveTabBackground: "var(--card)",
      editorActiveTabForeground: "var(--muted)",
      editorActiveTabBorderColor: "transparent",
      editorActiveTabIndicatorTopColor: "transparent",
      editorActiveTabIndicatorBottomColor: "var(--accent)",
      terminalTitlebarBackground: "var(--bg2)",
      terminalTitlebarBorderBottomColor: "var(--line)",
      terminalBackground: "var(--card)",
    },
  },
};
