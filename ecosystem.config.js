const path = require("path");

const binName = process.platform === "win32" ? "iran-discord-stats.exe" : "iran-discord-stats";
const discardOut = process.platform === "win32" ? "NUL" : "/dev/null";

module.exports = {
  apps: [
    {
      name: "IRAN-DISCORD-STATS",
      script: path.join("bin", binName),
      cwd: __dirname,
      interpreter: "none",
      windowsHide: true,
      autorestart: true,
      max_restarts: 20,
      restart_delay: 26000,
      watch: false,
      out_file: discardOut,
      error_file: path.join(__dirname, "logs", "err.log"),
      merge_logs: true,
    },
  ],
};
