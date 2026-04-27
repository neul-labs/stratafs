; AgentFS Windows Desktop Installer
; Requires NSIS 3.0 or later

!define APPNAME "AgentFS"
!define COMPANYNAME "AgentFS Team"
!define DESCRIPTION "The Agentic Filesystem for AI agents"
!define VERSIONMAJOR 0
!define VERSIONMINOR 2
!define VERSIONBUILD 0
!define VERSION "${VERSIONMAJOR}.${VERSIONMINOR}.${VERSIONBUILD}"

; These will be displayed by the "Click here for support information" link in "Add/Remove Programs"
!define HELPURL "https://github.com/yourusername/agentfs/issues"
!define UPDATEURL "https://github.com/yourusername/agentfs/releases"
!define ABOUTURL "https://github.com/yourusername/agentfs"

; This is the size (in kB) of all the files copied into "Program Files"
!define INSTALLSIZE 50000

; Include Modern UI
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"
!include "FileFunc.nsh"

; Request application privileges for Windows Vista/7/8/10
RequestExecutionLevel admin

; Best compression
SetCompressor /SOLID lzma

; Define installer details
Name "${APPNAME}"
Icon "agentfs.ico"
OutFile "AgentFS-${VERSION}-Setup.exe"
InstallDir "$PROGRAMFILES64\${APPNAME}"
InstallDirRegKey HKLM "Software\${COMPANYNAME}\${APPNAME}" "Install_Dir"

; Interface settings
!define MUI_ABORTWARNING
!define MUI_ICON "agentfs.ico"
!define MUI_UNICON "agentfs.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "wizard.bmp"

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE.txt"
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY

; Custom page for service configuration
Page custom ServiceConfigPage ServiceConfigPageLeave

!insertmacro MUI_PAGE_INSTFILES

; Finish page options
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Start AgentFS Desktop"
!define MUI_FINISHPAGE_RUN_FUNCTION "LaunchAgentFS"
!define MUI_FINISHPAGE_SHOWREADME "$INSTDIR\README.txt"
!define MUI_FINISHPAGE_LINK "Visit AgentFS website"
!define MUI_FINISHPAGE_LINK_LOCATION "${ABOUTURL}"

!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Languages
!insertmacro MUI_LANGUAGE "English"

; Version information
VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "${APPNAME}"
VIAddVersionKey "CompanyName" "${COMPANYNAME}"
VIAddVersionKey "FileDescription" "${DESCRIPTION}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "LegalCopyright" "© ${COMPANYNAME}"

; Global variables
Var StartMenuFolder
Var CreateDesktopShortcut
Var CreateStartMenuShortcut
Var InstallAsService
Var AutoStart

; Initialization
Function .onInit
    ; Check if already installed
    ReadRegStr $R0 HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "UninstallString"
    StrCmp $R0 "" done

    MessageBox MB_OKCANCEL|MB_ICONEXCLAMATION \
        "${APPNAME} is already installed. $\n$\nClick 'OK' to remove the previous version or 'Cancel' to cancel this upgrade." \
        IDOK uninst
    Abort

    uninst:
        ClearErrors
        ExecWait '$R0 _?=$INSTDIR'

        IfErrors no_remove_uninstaller done
        no_remove_uninstaller:

    done:
        ; Initialize variables
        StrCpy $CreateDesktopShortcut ${BST_CHECKED}
        StrCpy $CreateStartMenuShortcut ${BST_CHECKED}
        StrCpy $InstallAsService ${BST_UNCHECKED}
        StrCpy $AutoStart ${BST_CHECKED}
        StrCpy $StartMenuFolder "AgentFS"
FunctionEnd

; Custom service configuration page
Function ServiceConfigPage
    !insertmacro MUI_HEADER_TEXT "Service Configuration" "Choose how to run AgentFS"

    nsDialogs::Create 1018
    Pop $0

    ${NSD_CreateLabel} 0 0 100% 20u "AgentFS can run as a Windows service or as a desktop application:"

    ${NSD_CreateRadioButton} 20 30 280u 15u "Run as desktop application (recommended)"
    Pop $1
    ${NSD_Check} $1

    ${NSD_CreateRadioButton} 20 50 280u 15u "Run as Windows service"
    Pop $2

    ${NSD_CreateLabel} 40 70 260u 30u "Desktop application: Easy to use, runs when you're logged in"

    ${NSD_CreateLabel} 40 100 260u 30u "Windows service: Runs in background, always available"

    ${NSD_CreateCheckbox} 0 140 100% 15u "Start AgentFS automatically"
    Pop $3
    ${NSD_Check} $3

    ${NSD_CreateCheckbox} 0 160 100% 15u "Create desktop shortcut"
    Pop $4
    ${NSD_Check} $4

    ${NSD_CreateCheckbox} 0 180 100% 15u "Create Start Menu shortcuts"
    Pop $5
    ${NSD_Check} $5

    nsDialogs::Show
FunctionEnd

Function ServiceConfigPageLeave
    ${NSD_GetState} $2 $InstallAsService
    ${NSD_GetState} $3 $AutoStart
    ${NSD_GetState} $4 $CreateDesktopShortcut
    ${NSD_GetState} $5 $CreateStartMenuShortcut
FunctionEnd

; Installation sections
Section "AgentFS Core" SecCore
    SectionIn RO ; Required

    SetOutPath $INSTDIR

    ; Main executable and libraries
    File "agentfs.exe"
    File "onnxruntime.dll"
    File "README.txt"
    File "LICENSE.txt"

    ; Configuration
    SetOutPath $INSTDIR\config
    File /r "config\*.*"

    ; Create data directory
    CreateDirectory "$INSTDIR\data"

    ; Create user data directory
    CreateDirectory "$APPDATA\AgentFS"

    ; Write registry entries
    WriteRegStr HKLM "Software\${COMPANYNAME}\${APPNAME}" "Install_Dir" "$INSTDIR"
    WriteRegStr HKLM "Software\${COMPANYNAME}\${APPNAME}" "Version" "${VERSION}"

    ; Write uninstall information
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayName" "${APPNAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "InstallLocation" "$INSTDIR"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayIcon" "$INSTDIR\agentfs.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "Publisher" "${COMPANYNAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "HelpLink" "${HELPURL}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "URLUpdateInfo" "${UPDATEURL}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "URLInfoAbout" "${ABOUTURL}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "DisplayVersion" "${VERSION}"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "VersionMajor" ${VERSIONMAJOR}
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "VersionMinor" ${VERSIONMINOR}
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "NoRepair" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" "EstimatedSize" ${INSTALLSIZE}

    ; Create uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    ; Initialize configuration
    ExecWait '"$INSTDIR\agentfs.exe" config init --config-dir="$APPDATA\AgentFS"'
SectionEnd

Section "Desktop Integration" SecDesktop
    ; Create shortcuts if requested
    ${If} $CreateDesktopShortcut == ${BST_CHECKED}
        CreateShortcut "$DESKTOP\AgentFS.lnk" "$INSTDIR\agentfs.exe" "" "$INSTDIR\agentfs.exe" 0
    ${EndIf}

    ${If} $CreateStartMenuShortcut == ${BST_CHECKED}
        CreateDirectory "$SMPROGRAMS\$StartMenuFolder"
        CreateShortcut "$SMPROGRAMS\$StartMenuFolder\AgentFS.lnk" "$INSTDIR\agentfs.exe" "" "$INSTDIR\agentfs.exe" 0
        CreateShortcut "$SMPROGRAMS\$StartMenuFolder\AgentFS Configuration.lnk" "$INSTDIR\agentfs.exe" "config show" "$INSTDIR\agentfs.exe" 0
        CreateShortcut "$SMPROGRAMS\$StartMenuFolder\Uninstall AgentFS.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0
    ${EndIf}
SectionEnd

Section /o "Windows Service" SecService
    ; Install as Windows service
    ExecWait '"$INSTDIR\agentfs.exe" service install --config-dir="$APPDATA\AgentFS"'

    ${If} $AutoStart == ${BST_CHECKED}
        ; Start service
        ExecWait 'sc start AgentFS'

        ; Set service to auto-start
        ExecWait 'sc config AgentFS start= auto'
    ${EndIf}
SectionEnd

Section "Auto-Start" SecAutoStart
    ${If} $InstallAsService != ${BST_CHECKED}
    ${AndIf} $AutoStart == ${BST_CHECKED}
        ; Add to startup registry
        WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "AgentFS" "$INSTDIR\agentfs.exe"
    ${EndIf}
SectionEnd

Section "Visual C++ Redistributable" SecVCRedist
    ; Download and install VC++ Redistributable if needed
    SetOutPath $TEMP
    NSISdl::download "https://aka.ms/vs/17/release/vc_redist.x64.exe" "vc_redist.x64.exe"
    ExecWait "$TEMP\vc_redist.x64.exe /quiet /norestart"
    Delete "$TEMP\vc_redist.x64.exe"
SectionEnd

; Section descriptions
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "Core AgentFS application and libraries"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "Desktop shortcuts and Start Menu integration"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecService} "Install AgentFS as a Windows service"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecAutoStart} "Start AgentFS automatically when Windows starts"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecVCRedist} "Microsoft Visual C++ Redistributable (required)"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; Functions
Function LaunchAgentFS
    ${If} $InstallAsService == ${BST_CHECKED}
        ExecShell "" "sc" "start AgentFS"
    ${Else}
        Exec "$INSTDIR\agentfs.exe"
    ${EndIf}
FunctionEnd

; Uninstaller
Section "Uninstall"
    ; Stop service if running
    ExecWait 'sc stop AgentFS'
    ExecWait '"$INSTDIR\agentfs.exe" service uninstall'

    ; Remove from startup
    DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "AgentFS"

    ; Remove files
    Delete "$INSTDIR\agentfs.exe"
    Delete "$INSTDIR\onnxruntime.dll"
    Delete "$INSTDIR\README.txt"
    Delete "$INSTDIR\LICENSE.txt"
    Delete "$INSTDIR\uninstall.exe"

    ; Remove config directory
    RMDir /r "$INSTDIR\config"
    RMDir /r "$INSTDIR\data"

    ; Remove shortcuts
    Delete "$DESKTOP\AgentFS.lnk"
    Delete "$SMPROGRAMS\$StartMenuFolder\AgentFS.lnk"
    Delete "$SMPROGRAMS\$StartMenuFolder\AgentFS Configuration.lnk"
    Delete "$SMPROGRAMS\$StartMenuFolder\Uninstall AgentFS.lnk"
    RMDir "$SMPROGRAMS\$StartMenuFolder"

    ; Remove registry entries
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}"
    DeleteRegKey HKLM "Software\${COMPANYNAME}\${APPNAME}"
    DeleteRegKey /ifempty HKLM "Software\${COMPANYNAME}"

    ; Remove installation directory
    RMDir "$INSTDIR"

    ; Ask about user data
    MessageBox MB_YESNO "Do you want to remove user data and configuration files?" IDNO skip_userdata
        RMDir /r "$APPDATA\AgentFS"
    skip_userdata:
SectionEnd