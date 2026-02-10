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

// Helper to extract values from our simple config.json
function GetJSONValue(const JSON, Key: String): String;
var
  KeyPos, ValueStart, ValueEnd: Integer;
  SearchKey: String;
begin
  Result := '';
  SearchKey := '"' + Key + '":"';
  KeyPos := Pos(SearchKey, JSON);
  if KeyPos > 0 then
  begin
    ValueStart := KeyPos + Length(SearchKey);
    ValueEnd := Pos('"', Copy(JSON, ValueStart, MaxInt));
    if ValueEnd > 0 then
      Result := Copy(JSON, ValueStart, ValueEnd - 1);
  end;
end;

procedure StopSloPNProcesses();
var
  ResultCode: Integer;
begin
  Exec('taskkill.exe', '/F /IM {#MyAppExeName} /T', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec('sc.exe', 'stop SloPNHelper', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Exec('taskkill.exe', '/F /IM {#MyHelperExeName} /T', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Sleep(1000);
end;

procedure InitializeWizard;
var
  OldConfig: String;
  ConfigPath: String;
begin
  ConfigPage := CreateInputQueryPage(wpReady,
    'SloPN Configuration', 'Server Connection Details',
    'Please enter the connection details provided by your SloPN server administrator.');
  ConfigPage.Add('Server Address (e.g. 1.2.3.4:4242):', False);
  ConfigPage.Add('Auth Token:', True);
  
  // Try to load existing config to pre-fill
  ConfigPath := ExpandConstant('{userappdata}') + '\SloPN\config.json';
  if FileExists(ConfigPath) then
  begin
    if LoadStringFromFile(ConfigPath, OldConfig) then
    begin
      ConfigPage.Values[0] := GetJSONValue(OldConfig, 'server');
      ConfigPage.Values[1] := GetJSONValue(OldConfig, 'token');
    end;
  end;
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