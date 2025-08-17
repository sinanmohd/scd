{
  mkShell,
  scid,
  nixfmt-rfc-style,
  gopls,
}:

mkShell {
  inputsFrom = [ scid ];

  buildInputs = [
    gopls
    nixfmt-rfc-style
  ];

  shellHook = ''
    export PS1="\033[0;31m[scid]\033[0m $PS1"
  '';
}
