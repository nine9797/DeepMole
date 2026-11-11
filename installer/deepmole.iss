#ifndef AppVersion
  #define AppVersion "dev"
#endif
#ifndef ZapretDir
  #define ZapretDir "..\dist\zapret"
#endif

[Setup]
AppId={{7E3F2C1A-5B8D-4A96-9C0E-2D4F6A8B1C35}
AppName=DeepMole
AppVersion={#AppVersion}
AppVerName=DeepMole {#AppVersion}
DefaultDirName={autopf}\DeepMole
DefaultGroupName=DeepMole
DisableProgramGroupPage=yes
PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir=..\dist
OutputBaseFilename=DeepMole-{#AppVersion}-setup
SetupIconFile=..\assets\icon.ico
UninstallDisplayIcon={app}\deepmole.exe
UninstallDisplayName=DeepMole
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
CloseApplications=force
RestartApplications=no

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"

[CustomMessages]
english.Autostart=Start DeepMole when you sign in to Windows
russian.Autostart=Запускать DeepMole при входе в Windows
english.Extra=Additional options:
russian.Extra=Дополнительно:
english.Uninstall=Uninstall DeepMole
russian.Uninstall=Удалить DeepMole
english.RunApp=Launch DeepMole
russian.RunApp=Запустить DeepMole

[Tasks]
Name: "autostart"; Description: "{cm:Autostart}"; GroupDescription: "{cm:Extra}"
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "..\dist\deepmole.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\deepmole-cli.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\deepmole.example.json"; DestDir: "{app}"; DestName: "deepmole.json"; Flags: onlyifdoesntexist uninsneveruninstall
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.en.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ZapretDir}\*"; DestDir: "{app}\zapret"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\DeepMole"; Filename: "{app}\deepmole.exe"
Name: "{group}\{cm:Uninstall}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\DeepMole"; Filename: "{app}\deepmole.exe"; Tasks: desktopicon

[Run]
Filename: "schtasks.exe"; Parameters: "/Create /F /TN DeepMole /SC ONLOGON /RL HIGHEST /TR ""\""{app}\deepmole.exe\"""""; Flags: runhidden; Tasks: autostart
Filename: "{app}\deepmole.exe"; Description: "{cm:RunApp}"; Flags: nowait postinstall skipifsilent runascurrentuser

[UninstallRun]
Filename: "taskkill.exe"; Parameters: "/F /IM deepmole.exe"; Flags: runhidden; RunOnceId: "killapp"
Filename: "taskkill.exe"; Parameters: "/F /IM winws.exe"; Flags: runhidden; RunOnceId: "killwinws"
Filename: "schtasks.exe"; Parameters: "/Delete /F /TN DeepMole"; Flags: runhidden; RunOnceId: "task"
Filename: "sc.exe"; Parameters: "stop WinDivert"; Flags: runhidden; RunOnceId: "wdstop"
Filename: "sc.exe"; Parameters: "delete WinDivert"; Flags: runhidden; RunOnceId: "wddel"
Filename: "sc.exe"; Parameters: "stop WinDivert14"; Flags: runhidden; RunOnceId: "wd14stop"
Filename: "sc.exe"; Parameters: "delete WinDivert14"; Flags: runhidden; RunOnceId: "wd14del"

[UninstallDelete]
Type: files; Name: "{app}\deepmole.log"
Type: files; Name: "{app}\deepmole-state.json"
Type: files; Name: "{app}\deepmole.json"
Type: filesandordirs; Name: "{app}\zapret\lists"
Type: filesandordirs; Name: "{commonappdata}\DeepMole"

[Code]
procedure KillRunning();
var
  R: Integer;
begin
  Exec('taskkill.exe', '/F /IM deepmole.exe', '', SW_HIDE, ewWaitUntilTerminated, R);
  Exec('taskkill.exe', '/F /IM winws.exe', '', SW_HIDE, ewWaitUntilTerminated, R);
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
begin
  KillRunning();
  Result := '';
end;
