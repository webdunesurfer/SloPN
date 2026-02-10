#define MyAppName "SloPN"
#define MyAppVersion "0.5.3"
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
CloseApplications=force
RestartApplications=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
; GUI
Source: "..\..\bin\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
; Helper
Source: "..\..\bin\{#MyHelperExeName}"; DestDir: "{app}"; Flags: ignoreversion
; Drivers
Source: "driver\*"; DestDir: "{app}\driver"; Flags: ignoreversion recursesubdirs

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

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

procedure StopSloPNProcesses();
var
  ResultCode: Integer;
begin
  // Force kill the GUI first to release file handles
  Exec('taskkill.exe', '/F /IM {#MyAppExeName} /T', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  
  // Try to stop the service gracefully
  Exec('sc.exe', 'stop SloPNHelper', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  
  // Force kill the helper process just in case the service is stuck
  Exec('taskkill.exe', '/F /IM {#MyHelperExeName} /T', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  
  // Give Windows a moment to release all file locks
  Sleep(1000);
end;

procedure InitializeWizard;
begin
  ConfigPage := CreateInputQueryPage(wpReady,
    'SloPN Configuration', 'Server Connection Details',
    'Please enter the connection details provided by your SloPN server administrator.');
  ConfigPage.Add('Server Address (e.g. 1.2.3.4:4242):', False);
  ConfigPage.Add('Auth Token:', True); // Password/Hidden field
  
  ConfigPage.Values[0] := '';
  ConfigPage.Values[1] := '';
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  SettingsPath: String;
  SettingsContent: String;
  ServerVal: String;
  TokenVal: String;
begin
  if CurStep = ssInstall then
  begin
    StopSloPNProcesses();
  end;

  if CurStep = ssPostInstall then
  begin
    ServerVal := ConfigPage.Values[0];
    TokenVal := ConfigPage.Values[1];
    
    SettingsPath := ExpandConstant('{userappdata}') + '\SloPN';
    ForceDirectories(SettingsPath);
    
    SettingsContent := '{' + #13#10 +
      '  "server": "' + ServerVal + '",' + #13#10 +
      '  "full_tunnel": true' + #13#10 +
      '}';
    SaveStringToFile(SettingsPath + '\settings.json', SettingsContent, False);
    SaveStringToFile(SettingsPath + '\config.json', '{"server":"' + ServerVal + '", "token":"' + TokenVal + '"}', False);
  end;
end;
