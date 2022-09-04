var binwrap = require("binwrap");
var path = require("path");

var packageInfo = require(path.join(__dirname, "package.json"));
var version = packageInfo.version;
var root = "https://github.com/TekWizely/run/releases/download/v" + version
module.exports = binwrap({
  dirname: __dirname,
  binaries: [
    "run"
  ],
  urls: {
    "darwin-x64": root + "/run_" + version + "_darwin_amd64.tar.gz",
    "darwin-arm64": root + "/run_" + version + "_darwin_arm64.tar.gz",
    "linux-x64": root + "/run_" + version + "_linux_amd64.tar.gz"
  }
});
