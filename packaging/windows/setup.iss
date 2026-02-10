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
Source: "..\..\bin\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
; Helper
Source: "..\..\bin\{#MyHelperExeName}"; DestDir: "{app}"; Flags: ignoreversion; BeforeInstall: StopSloPNService
; Drivers
Source: "driver\*"; DestDir: "{app}\driver"; Flags: ignoreversion recursesubdirs

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autoprodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
; Install TAP Driver
Filename: "{app}\driver\tapinstall.exe"; Parameters: "install ""{app}\driver\OemVista.inf"" tap0901"; StatusMsg: "Installing virtual network adapter..."; Flags: runhidden
; Create/Update Helper Service
Filename: "{sys}\sc.exe"; Parameters: "create SloPNHelper binPath= ""{app}\{#MyHelperExeName}"" start= auto displayname= ""SloPN Privileged Helper"""; Flags: runhidden
; Ensure binPath is updated if it changed
Filename: "{sys}\sc.exe"; Parameters: "config SloPNHelper binPath= ""{app}\{#MyHelperExeName}"""; Flags: runhidden
Filename: "{sys}\sc.exe"; Parameters: "start SloPNHelper"; Flags: runhidden
; Launch GUI
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
; Stop and Delete Service
Filename: "{sys}\sc.exe"; Parameters: "stop SloPNHelper"; Flags: runhidden
Filename: "{sys}\sc.exe"; Parameters: "delete SloPNHelper"; Flags: runhidden
; Remove TAP Driver
Filename: "{app}\driver\tapinstall.exe"; Parameters: "remove tap0901"; Flags: runhidden

[Code]
var
  ConfigPage: TInputQueryWizardPage;

procedure StopSloPNService();
var
  ResultCode: Integer;
begin
  // Try to stop the service before overwriting files
  Exec('sc.exe', 'stop SloPNHelper', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  // Give it a moment to release file handles
  Sleep(1000);
end;

function InitializeSetup(): Boolean;
begin
  Randomize;
  Result := True;
end;

procedure InitializeWizard;
begin
  ConfigPage := CreateInputQueryPage(wpReady,
    'SloPN Configuration', 'Server Connection Details',
    'Please enter the connection details provided by your SloPN server administrator.');
  ConfigPage.Add('Server Address (e.g. 1.2.3.4:4242):', False);
  ConfigPage.Add('Auth Token:', True); // Password/Hidden field
  
  // Set default values if needed
  ConfigPage.Values[0] := '';
  ConfigPage.Values[1] := '';
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  Secret: String;
  SecretPath: String;
  SettingsPath: String;
  SettingsContent: String;
  ServerVal: String;
  TokenVal: String;
begin
  if CurStep = ssPostInstall then
  begin
    // 1. Generate IPC secret if it doesn't exist
    SecretPath := 'C:\ProgramData\SloPN\ipc.secret';
    if not FileExists(SecretPath) then
    begin
      ForceDirectories('C:\ProgramData\SloPN');
      Secret := IntToHex(Random(2147483647), 8) + IntToHex(Random(2147483647), 8);
      SaveStringToFile(SecretPath, Secret, False);
    end;

    // 2. Save User Configuration from Wizard
    ServerVal := ConfigPage.Values[0];
    TokenVal := ConfigPage.Values[1];
    
    // We save this to %APPDATA%\SloPN\settings.json
    SettingsPath := ExpandConstant('{userappdata}') + '\SloPN';
    ForceDirectories(SettingsPath);
    
    // Create a simple JSON structure
    SettingsContent := '{' + #13#10 +
      '  "server": "' + ServerVal + '",' + #13#10 +
      '  "full_tunnel": false' + #13#10 +
      '}';
    SaveStringToFile(SettingsPath + '\settings.json', SettingsContent, False);

    // Also save the token for the GUI to pick up
    SaveStringToFile(SettingsPath + '\config.json', '{"server":"' + ServerVal + '", "token":"' + TokenVal + '"}', False);
  end;
end;