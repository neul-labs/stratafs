; StrataFS Windows Installer
; Build with: makensis installer.nsi

!include "MUI2.nsh"
!include "FileFunc.nsh"

; Installer attributes
Name "StrataFS"
OutFile "..\..\..\build\windows\StrataFS-Setup.exe"
InstallDir "$PROGRAMFILES64\StrataFS"
InstallDirRegKey HKLM "Software\StrataFS" "InstallPath"
RequestExecutionLevel admin

; Version info
!define VERSION "0.2.0"
VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "StrataFS"
VIAddVersionKey "CompanyName" "StrataFS"
VIAddVersionKey "FileDescription" "StrataFS Installer"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"

; Modern UI settings
!define MUI_ABORTWARNING
!define MUI_ICON "..\resources\stratafs.ico"
!define MUI_UNICON "..\resources\stratafs.ico"

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
Section "StrataFS Core" SecCore
    SectionIn RO ; Required

    SetOutPath "$INSTDIR"

    ; Copy main files
    File "..\..\..\build\windows\stratafs.exe"
    File "..\..\..\build\windows\stratafs-ui.exe"
    File "..\..\..\build\windows\stratafs-service.exe"
    File "..\..\..\build\windows\stratafs-tray.exe"

    ; Copy ONNX runtime
    File "..\..\..\build\windows\onnxruntime.dll"

    ; Copy shell extensions
    File "..\..\..\build\windows\StrataFSContextMenu.dll"
    File "..\..\..\build\windows\StrataFSFilter.dll"

    ; Create data directory
    CreateDirectory "$PROFILE\.stratafs"

    ; Write registry keys
    WriteRegStr HKLM "Software\StrataFS" "InstallPath" "$INSTDIR"
    WriteRegStr HKLM "Software\StrataFS" "Version" "${VERSION}"

    ; Create uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Add to Add/Remove Programs
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\StrataFS" \
        "DisplayName" "StrataFS"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\StrataFS" \
        "UninstallString" "$\"$INSTDIR\Uninstall.exe$\""
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\StrataFS" \
        "DisplayIcon" "$INSTDIR\stratafs.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\StrataFS" \
        "Publisher" "StrataFS"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\StrataFS" \
        "DisplayVersion" "${VERSION}"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\StrataFS" \
        "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\StrataFS" \
        "NoRepair" 1

    ; Get install size
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\StrataFS" \
        "EstimatedSize" "$0"
SectionEnd

Section "Windows Service" SecService
    ; Install and start service
    nsExec::ExecToLog '"$INSTDIR\stratafs-service.exe" install'
    nsExec::ExecToLog '"$INSTDIR\stratafs-service.exe" start'
SectionEnd

Section "Shell Integration" SecShell
    ; Register context menu extension
    RegDLL "$INSTDIR\StrataFSContextMenu.dll"

    ; Register IFilter
    RegDLL "$INSTDIR\StrataFSFilter.dll"

    ; Import registry entries
    nsExec::ExecToLog 'regedit /s "$INSTDIR\StrataFSFilter.reg"'
    nsExec::ExecToLog 'regedit /s "$INSTDIR\StrataFSContextMenu.reg"'
SectionEnd

Section "Start Menu Shortcuts" SecShortcuts
    CreateDirectory "$SMPROGRAMS\StrataFS"
    CreateShortcut "$SMPROGRAMS\StrataFS\StrataFS.lnk" "$INSTDIR\stratafs-ui.exe"
    CreateShortcut "$SMPROGRAMS\StrataFS\Uninstall.lnk" "$INSTDIR\Uninstall.exe"
SectionEnd

Section "Desktop Shortcut" SecDesktop
    CreateShortcut "$DESKTOP\StrataFS.lnk" "$INSTDIR\stratafs-ui.exe"
SectionEnd

Section "Start with Windows" SecAutostart
    WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" \
        "StrataFS" "$INSTDIR\stratafs-tray.exe"
SectionEnd

; Section descriptions
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "Core StrataFS files (required)"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecService} "Install StrataFS as a Windows Service"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecShell} "Add context menu and search integration"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecShortcuts} "Create Start Menu shortcuts"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "Create Desktop shortcut"
    !insertmacro MUI_DESCRIPTION_TEXT ${SecAutostart} "Start StrataFS tray app when Windows starts"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; Uninstaller
Section "Uninstall"
    ; Stop and remove service
    nsExec::ExecToLog '"$INSTDIR\stratafs-service.exe" stop'
    nsExec::ExecToLog '"$INSTDIR\stratafs-service.exe" remove'

    ; Unregister shell extensions
    UnRegDLL "$INSTDIR\StrataFSContextMenu.dll"
    UnRegDLL "$INSTDIR\StrataFSFilter.dll"

    ; Remove files
    Delete "$INSTDIR\stratafs.exe"
    Delete "$INSTDIR\stratafs-ui.exe"
    Delete "$INSTDIR\stratafs-service.exe"
    Delete "$INSTDIR\stratafs-tray.exe"
    Delete "$INSTDIR\onnxruntime.dll"
    Delete "$INSTDIR\StrataFSContextMenu.dll"
    Delete "$INSTDIR\StrataFSFilter.dll"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir "$INSTDIR"

    ; Remove shortcuts
    Delete "$SMPROGRAMS\StrataFS\StrataFS.lnk"
    Delete "$SMPROGRAMS\StrataFS\Uninstall.lnk"
    RMDir "$SMPROGRAMS\StrataFS"
    Delete "$DESKTOP\StrataFS.lnk"

    ; Remove registry entries
    DeleteRegKey HKLM "Software\StrataFS"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\StrataFS"
    DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "StrataFS"
SectionEnd
