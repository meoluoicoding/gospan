; Gospan Windows installer (Inno Setup)
; Build with: iscc scripts/installer.iss
; Requires: Inno Setup 6+ (https://jrsoftware.org/isinfo.php)

#define MyAppName "Gospan"
#define MyAppVersion "0.1.0"
#define MyAppPublisher "Gospan"
#define MyAppURL "https://github.com/meoluoicoding/gospan"
#define MyAppExeName "gospan.exe"

[Setup]
AppId={{B4F3C2A1-7E8D-4F9A-B2C3-D4E5F6A7B8C9}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}/releases
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
LicenseFile=..\LICENSE
OutputDir=..\dist
OutputBaseFilename=gospan-{#MyAppVersion}-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=lowest
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "addtopath"; Description: "Add Gospan to PATH"; GroupDescription: "Additional options:"

[Files]
Source: "..\dist\gospan.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\gospan-gui.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\configs\config.example.yaml"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion isreadme

[Icons]
Name: "{group}\Gospan GUI"; Filename: "{app}\gospan-gui.exe"
Name: "{group}\Gospan CLI"; Filename: "{app}\gospan.exe"
Name: "{autodesktop}\Gospan"; Filename: "{app}\gospan-gui.exe"; Tasks: desktopicon

[Registry]
; Register gospan.exe on PATH (optional task)
Root: HKCU; Subkey: "Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Tasks: addtopath; Check: NeedsAddPath('{app}')

[Run]
Filename: "{app}\gospan.exe"; Parameters: "config init --force"; Flags: runhidden
Filename: "{app}\gospan-gui.exe"; Description: "{cm:LaunchProgram,Gospan}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{userappdata}\gospan"

[Code]
function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', OrigPath) then
    Result := True
  else
    Result := Pos(';' + Param + ';', ';' + OrigPath + ';') = 0;
end;
