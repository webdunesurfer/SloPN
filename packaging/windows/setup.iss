#define MyAppName "SloPN"
#define MyAppVersion "0.5.1"
#define MyAppPublisher "webdunesurfer"
#define MyAppURL "https://github.com/webdunesurfer/SloPN"
#define MyAppExeName "SloPN.exe"
#define MyHelperExeName "slopn-helper.exe"

[Setup]
AppId={{C78A9C12-3D4F-4E5G-6H7I-8J9K0L1M2N3O}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DisableProgramGroupPage=yes
OutputDir=..\..\bin
OutputBaseFilename=SloPN-Setup
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
; GUI
Source: "..\..\gui\build\bin\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
; Helper
Source: "..\..\slopn-helper.exe"; DestDir: "{app}"; Flags: ignoreversion
; Drivers
Source: "driver\*"; DestDir: "{app}\driver"; Flags: ignoreversion recursesubdirs

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
; Install TAP Driver
Filename: "{app}\driver	apinstall.exe"; Parameters: "install ""{app}\driver\OemVista.inf"" tap0901"; StatusMsg: "Installing virtual network adapter..."; Flags: runhidden
; Create Helper Service
Filename: "{sys}\sc.exe"; Parameters: "create SloPNHelper binPath= ""{app}\{#MyHelperExeName}"" start= auto displayname= ""SloPN Privileged Helper"""; Flags: runhidden
Filename: "{sys}\sc.exe"; Parameters: "start SloPNHelper"; Flags: runhidden
; Launch GUI
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
; Stop and Delete Service
Filename: "{sys}\sc.exe"; Parameters: "stop SloPNHelper"; Flags: runhidden
Filename: "{sys}\sc.exe"; Parameters: "delete SloPNHelper"; Flags: runhidden
; Remove TAP Driver
Filename: "{app}\driver	apinstall.exe"; Parameters: "remove tap0901"; Flags: runhidden

[Code]
function InitializeSetup(): Boolean;
begin
  Randomize;
  Result := True;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  Secret: String;
  SecretPath: String;
begin
  if CurStep = ssPostInstall then
  begin
    // Generate IPC secret if it doesn't exist
    SecretPath := 'C:\ProgramData\SloPN\ipc.secret';
    if not FileExists(SecretPath) then
    begin
      ForceDirectories('C:\ProgramData\SloPN');
      // Simple random-ish string for now (Inno doesn't have crypto rand)
      Secret := IntToHex(Random(2147483647), 8) + IntToHex(Random(2147483647), 8);
      SaveStringToFile(SecretPath, Secret, False);
    end;
  end;
end;
