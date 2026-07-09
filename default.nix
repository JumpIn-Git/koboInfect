{
  lib,
  buildGoModule,
  fetchFromGitHub,
  nix-update-script,
}:
buildGoModule (finalAttrs: {
  pname = "kobo-infect";
  version = "1.4.0";
  __structuredAttrs = true;

  src = fetchFromGitHub {
    owner = "JumpIn-Git";
    repo = "kobo-infect";
    tag = "v${finalAttrs.version}";
    hash = "sha256-7ZBDOwMhlFm0w/QhA5tss5lFFkhIPJ0KohQuEt5RC/o=";
  };

  vendorHash = "sha256-stU3ML9x1VP8/NXfGHcjyf6xt85TmhGWjW885IpYeZY=";

  ldflags = ["-s"];

  passthru.updateScript = nix-update-script {};

  meta = {
    description = "";
    homepage = "https://github.com/JumpIn-Git/kobo-infect";
    changelog = "https://github.com/JumpIn-Git/kobo-infect/releases/tag/${finalAttrs.src.tag}";
    license = lib.licenses.gpl3;
    maintainers = with lib.maintainers; [];
    mainProgram = "kobo-infect";
  };
})
