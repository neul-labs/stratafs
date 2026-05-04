; AgentFS Windows Installer
; Build with: makensis installer.nsi

!include "MUI2.nsh"
!include "FileFunc.nsh"

; Installer attributes
Name "AgentFS"
OutFile "..\..\..\build\windows\AgentFS-Setup.exe"
InstallDir "$PROGRAMFILES64\AgentFS"
InstallDirRegKey HKLM "Software\AgentFS" "InstallPath"
RequestExecutionLevel admin

; Version info
!define VERSION "0.2.0"
VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "AgentFS"
VIAddVersionKey "CompanyName" "AgentFS"
VIAddVersionKey "FileDescription" "AgentFS Installer"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"

; Modern UI settings
!define MUI_ABORTWARNING
!define MUI_ICON "..\resources\agentfs.ico"
!define MUI_UNICON "..\resources\agentfs.ico"

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\..\..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

; Sections
Section "AgentFS Core" SecCore
    SectionIn RO ; Required

    SetOutPath "$INSTDIR"

    ; Copy main files
    File "..\..\..\build\windows\agentfs.exe"
    File "..\..\..\build\windows\agentfs-ui.exe"
    File "..\..\..\build\windows\agentfs-service.exe"
    File "..\..\..\build\windows\agentfs-tray.exe"

    ; Copy ONNX runtime
    File "..\..\..\build\windows\onnxruntime.dll"

    ; Copy shell extensions
    File "..\..\..\build\windows\AgentFSContextMenu.dll"
    File "..\..\..\build\windows\AgentFSFilter.dll"

    ; Create data directory
    CreateDirectory "$PROFILE\.agentfs"

    ; Write registry keys
    WriteRegStr HKLM "Software\AgentFS" "InstallPath" "$INSTDIR"
    WriteRegStr HKLM "Software\AgentFS" "Version" "${VERSION}"

    ; Create uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Add to Add/Remove Programs
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AgentFS" \
        "DisplayName" "AgentFS"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AgentFS" \
        "UninstallString" "$\"$INSTDIR\Uninstall.exe$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AgentFS" \
        "DisplayIcon" "$INSTDIR\agentfs.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AgentFS" \
        "Publisher" "AgentFS"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AgentFS" \
        "DisplayVersion" "${VERSION}"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AgentFS" \
        "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AgentFS" \
        "NoRepair" 1

    ; Get install size
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AgentFS" \
        "EstimatedSize" "$0"
SectionEnd

Section "Windows Service" SecService
    ; Install and start service
    nsExec::ExecToLog '"$INSTDIR\agentfs-service.exe" install'
    nsExec::ExecToLog '"$INSTDIR\agentfs-service.exe" start'
SectionEnd

Section "Shell Integration" SecShell
    ; Register context menu extension
    RegDLL "$INSTDIR\AgentFSContextMenu.dll"

    ; Register IFilter
    RegDLL "$INSTDIR\AgentFSFilter.dll"

    ; Import registry entries
    nsExec::ExecToLog 'regedit /s "$INSTDIR\AgentFSFilter.reg"'
    nsExec::ExecToLog 'regedit /s "$INSTDIR\AgentFSContextMenu.reg"'
SectionEnd

Section "Start Menu Shortcuts" SecShortcuts
    CreateDirectory "$SMPROGRAMS\AgentFS"
    CreateShortcut "$SMPROGRAMS\AgentFS\AgentFS.lnk" "$INSTDIR\agentfs-ui.exe"
    CreateShortcut "$SMPROGRAMS\AgentFS\Uninstall.lnk" "$INSTDIR\Uninstall.exe"
SectionEnd

Section "Desktop Shortcut" SecDesktop
    CreateShortcut "$DESKTOP\AgentFS.lnk" "$INSTDIR\agentfs-ui.exe"
SectionEnd

Section "Start with Windows" SecAutostart
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" \
        "AgentFS" "$INSTDIR\agentfs-tray.exe"
SectionEnd

; Section descriptions
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "Core AgentFS files (required)"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecService} "Install AgentFS as a Windows Service"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecShell} "Add context menu and search integration"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecShortcuts} "Create Start Menu shortcuts"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "Create Desktop shortcut"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecAutostart} "Start AgentFS tray app when Windows starts"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; Uninstaller
Section "Uninstall"
    ; Stop and remove service
    nsExec::ExecToLog '"$INSTDIR\agentfs-service.exe" stop'
    nsExec::ExecToLog '"$INSTDIR\agentfs-service.exe" remove'

    ; Unregister shell extensions
    UnRegDLL "$INSTDIR\AgentFSContextMenu.dll"
    UnRegDLL "$INSTDIR\AgentFSFilter.dll"

    ; Remove files
    Delete "$INSTDIR\agentfs.exe"
    Delete "$INSTDIR\agentfs-ui.exe"
    Delete "$INSTDIR\agentfs-service.exe"
    Delete "$INSTDIR\agentfs-tray.exe"
    Delete "$INSTDIR\onnxruntime.dll"
    Delete "$INSTDIR\AgentFSContextMenu.dll"
    Delete "$INSTDIR\AgentFSFilter.dll"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir "$INSTDIR"

    ; Remove shortcuts
    Delete "$SMPROGRAMS\AgentFS\AgentFS.lnk"
    Delete "$SMPROGRAMS\AgentFS\Uninstall.lnk"
    RMDir "$SMPROGRAMS\AgentFS"
    Delete "$DESKTOP\AgentFS.lnk"

    ; Remove registry entries
    DeleteRegKey HKLM "Software\AgentFS"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\AgentFS"
    DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "AgentFS"
SectionEnd
