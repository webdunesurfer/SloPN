#define MyAppName "SloPN"
#define MyAppVersion "0.7.3"
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
; CLI
Source: "..\..\bin\slopn-cli.exe"; DestDir: "{app}"; Flags: ignoreversion
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

function IsVCRedistInstalled: Boolean;
var
  Installed: Cardinal;
begin
  // Check for Visual C++ 2015-2022 Redistributable (x64)
  Result := RegQueryDWordValue(HKEY_LOCAL_MACHINE, 'SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\x64', 'Installed', Installed) and (Installed = 1);
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
begin
  ConfigPage := CreateInputQueryPage(wpReady,
    'SloPN Configuration', 'Server Connection Details',
    'Please enter the connection details provided by your SloPN server administrator.');
  ConfigPage.Add('Server Address (e.g. 1.2.3.4:4242):', False);
  ConfigPage.Add('Auth Token:', True);
  
  ConfigPage.Values[0] := '';
  ConfigPage.Values[1] := '';
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
  VCRedistURL: String;
  VCRedistPath: String;
begin
  if not IsVCRedistInstalled then
  begin
    if MsgBox('SloPN requires Microsoft Visual C++ Redistributable to run correctly. Would you like to download and install it now?', mbConfirmation, MB_YESNO) = IDYES then
    begin
      VCRedistURL := 'https://aka.ms/vs/17/release/vc_redist.x64.exe';
      VCRedistPath := ExpandConstant('{tmp}\vc_redist.x64.exe');
      
      // Note: Inno Setup 6 doesn't have a built-in 'DownloadFile' Pascal function that works reliably without a plugin (IDP).
      // However, we can use PowerShell to download it.
      ExtractTemporaryFile('tapinstall.exe'); // Just a placeholder to ensure {tmp} exists
      
      if Exec('powershell.exe', '-ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri ''' + VCRedistURL + ''' -OutFile ''' + VCRedistPath + '''"', '', SW_HIDE, ewWaitUntilTerminated, ResultCode) and (ResultCode = 0) then
      begin
        if Exec(VCRedistPath, '/quiet /norestart', '', SW_SHOW, ewWaitUntilTerminated, ResultCode) then
        begin
          // Success
        end else
          MsgBox('Installation of Visual C++ Redistributable failed. You may need to install it manually.', mbError, MB_OK);
      end else
        MsgBox('Download of Visual C++ Redistributable failed. Please check your internet connection.', mbError, MB_OK);
    end;
  end;
  Result := '';
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
    
    // Ensure ProgramData directory exists for logs/secrets
    ForceDirectories(ExpandConstant('{commonappdata}') + '\SloPN');

    SettingsPath := ExpandConstant('{userappdata}') + '\SloPN';
    ForceDirectories(SettingsPath);
    
    // Write initial config (including token and obfuscate default) to ProgramData
    // This allows the GUI to pick it up via GetInitialConfig()
    SaveStringToFile(ExpandConstant('{commonappdata}') + '\SloPN\config.json', 
      '{"server":"' + ServerVal + '", "token":"' + TokenVal + '", "obfuscate": true}', False);

    // Create new install marker for GUI
    SaveStringToFile('C:\ProgramData\SloPN\.new_install', '', False);
  end;
end;