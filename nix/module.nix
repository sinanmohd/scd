inputs:
{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.services.scid;
  inherit (pkgs.stdenv.hostPlatform) system;

  configFormat = pkgs.formats.toml { };
  configFile = configFormat.generate "scid.toml" cfg.settings;

  defaultEnvs = {
    SCID_CONFIG = "${configFile}";
  };
in
{
  meta.maintainers = with lib.maintainers; [ sinanmohd ];

  options.services.scid = {
    enable = lib.mkEnableOption "scid";
    package = lib.mkOption {
      type = lib.types.package;
      description = "The scid package to use.";
      default = inputs.self.packages.${system}.scid;
    };

    environment = lib.mkOption {
      default = { };
      type = lib.types.attrsOf lib.types.str;
    };

    settings = lib.mkOption {
      inherit (configFormat) type;
      default = { };
      description = ''
        Configuration options for scid.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    environment.systemPackages = [ cfg.package ];

    # This service stores a potentially large amount of data.
    # Running it as a dynamic user would force chown to be run everytime the
    # service is restarted on a potentially large number of files.
    # That would cause unnecessary and unwanted delays.
    users = {
      groups.scid = { };
      users.scid = {
        isSystemUser = true;
        group = "scid";
      };
    };


  systemd =
    let
      name = "scid";
      meta.description = "Your frenly neighbourhood CI/CD.";
    in
    {
      timers.${name} = meta // {
        wantedBy = [ "timers.target" ];

        timerConfig = {
          OnCalendar = "*:0/1";
          Persistent = true;
        };
      };

      services.${name} = meta // rec {
        description = "";
        wantedBy = [ "multi-user.target" ];
        after = [
          "network-online.target"
        ] ++ lib.optional config.services.k3s.enable "k3s.service";
        wants = after;

        environment = defaultEnvs // cfg.environment;
        serviceConfig = {
          Type = "simple";
          Restart = "on-failure";

          StateDirectory = name;
          WorkingDirectory = "%S/${name}";

          ExecStart = lib.getExe cfg.package;
        };
      };
    };
  };
}
