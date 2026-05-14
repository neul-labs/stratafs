/*
 * StrataFS Windows Explorer Context Menu Extension
 *
 * Adds right-click context menu items to Windows Explorer.
 * Build with Visual Studio, register with regsvr32.
 */

#include <windows.h>
#include <shlobj.h>
#include <shlwapi.h>
#include <strsafe.h>
#include <string>
#include <vector>

// {B2C3D4E5-F6A7-8901-BCDE-F23456789012}
static const GUID CLSID_StrataFSContextMenu =
{ 0xb2c3d4e5, 0xf6a7, 0x8901, { 0xbc, 0xde, 0xf2, 0x34, 0x56, 0x78, 0x90, 0x12 } };

class StrataFSContextMenu : public IShellExtInit, public IContextMenu {
private:
    LONG m_refCount;
    std::vector<std::wstring> m_selectedFiles;
    bool m_isDirectory;

public:
    StrataFSContextMenu() : m_refCount(1), m_isDirectory(false) {}
    virtual ~StrataFSContextMenu() {}

    // IUnknown
    STDMETHODIMP QueryInterface(REFIID riid, void **ppv) {
        if (riid == IID_IUnknown) {
            *ppv = static_cast<IShellExtInit*>(this);
        } else if (riid == IID_IShellExtInit) {
            *ppv = static_cast<IShellExtInit*>(this);
        } else if (riid == IID_IContextMenu) {
            *ppv = static_cast<IContextMenu*>(this);
        } else {
            *ppv = nullptr;
            return E_NOINTERFACE;
        }
        AddRef();
        return S_OK;
    }

    STDMETHODIMP_(ULONG) AddRef() { return InterlockedIncrement(&m_refCount); }
    STDMETHODIMP_(ULONG) Release() {
        LONG ref = InterlockedDecrement(&m_refCount);
        if (ref == 0) delete this;
        return ref;
    }

    // IShellExtInit
    STDMETHODIMP Initialize(PCIDLIST_ABSOLUTE pidlFolder, IDataObject *pdtobj, HKEY hkeyProgID) {
        m_selectedFiles.clear();
        m_isDirectory = false;

        if (!pdtobj) return E_INVALIDARG;

        FORMATETC fe = { CF_HDROP, nullptr, DVASPECT_CONTENT, -1, TYMED_HGLOBAL };
        STGMEDIUM stm;

        if (FAILED(pdtobj->GetData(&fe, &stm))) {
            return E_INVALIDARG;
        }

        HDROP hDrop = static_cast<HDROP>(GlobalLock(stm.hGlobal));
        if (!hDrop) {
            ReleaseStgMedium(&stm);
            return E_INVALIDARG;
        }

        UINT fileCount = DragQueryFileW(hDrop, 0xFFFFFFFF, nullptr, 0);

        for (UINT i = 0; i < fileCount; i++) {
            wchar_t filePath[MAX_PATH];
            if (DragQueryFileW(hDrop, i, filePath, MAX_PATH)) {
                m_selectedFiles.push_back(filePath);

                // Check if first item is directory
                if (i == 0) {
                    DWORD attrs = GetFileAttributesW(filePath);
                    m_isDirectory = (attrs != INVALID_FILE_ATTRIBUTES) && (attrs & FILE_ATTRIBUTE_DIRECTORY);
                }
            }
        }

        GlobalUnlock(stm.hGlobal);
        ReleaseStgMedium(&stm);

        return m_selectedFiles.empty() ? E_INVALIDARG : S_OK;
    }

    // IContextMenu
    STDMETHODIMP QueryContextMenu(HMENU hmenu, UINT indexMenu, UINT idCmdFirst, UINT idCmdLast, UINT uFlags) {
        if (uFlags & CMF_DEFAULTONLY) return MAKE_HRESULT(SEVERITY_SUCCESS, 0, 0);

        // Create submenu
        HMENU hSubmenu = CreatePopupMenu();

        if (m_isDirectory) {
            // Directory menu items
            InsertMenuW(hSubmenu, 0, MF_BYPOSITION | MF_STRING, idCmdFirst + 0, L"Add to StrataFS");
            InsertMenuW(hSubmenu, 1, MF_BYPOSITION | MF_STRING, idCmdFirst + 1, L"Export Metadata Here");
        } else {
            // File menu items
            InsertMenuW(hSubmenu, 0, MF_BYPOSITION | MF_STRING, idCmdFirst + 0, L"View StrataFS Metadata");
            InsertMenuW(hSubmenu, 1, MF_BYPOSITION | MF_STRING, idCmdFirst + 1, L"View Chunks");
            InsertMenuW(hSubmenu, 2, MF_BYPOSITION | MF_SEPARATOR, 0, nullptr);
            InsertMenuW(hSubmenu, 3, MF_BYPOSITION | MF_STRING, idCmdFirst + 2, L"Find Similar Files");
            InsertMenuW(hSubmenu, 4, MF_BYPOSITION | MF_SEPARATOR, 0, nullptr);
            InsertMenuW(hSubmenu, 5, MF_BYPOSITION | MF_STRING, idCmdFirst + 3, L"Reindex");
        }

        // Insert submenu into context menu
        MENUITEMINFOW mii = { sizeof(mii) };
        mii.fMask = MIIM_SUBMENU | MIIM_STRING | MIIM_ID;
        mii.wID = idCmdFirst + 10;
        mii.hSubMenu = hSubmenu;
        mii.dwTypeData = const_cast<LPWSTR>(L"StrataFS");
        InsertMenuItemW(hmenu, indexMenu, TRUE, &mii);

        return MAKE_HRESULT(SEVERITY_SUCCESS, 0, 11);
    }

    STDMETHODIMP InvokeCommand(CMINVOKECOMMANDINFO *pici) {
        if (HIWORD(pici->lpVerb) != 0) return E_INVALIDARG;

        UINT cmd = LOWORD(pici->lpVerb);

        if (m_isDirectory) {
            switch (cmd) {
                case 0: return AddSource();
                case 1: return ExportMetadata();
            }
        } else {
            switch (cmd) {
                case 0: return ViewMetadata();
                case 1: return ViewChunks();
                case 2: return FindSimilar();
                case 3: return Reindex();
            }
        }

        return E_INVALIDARG;
    }

    STDMETHODIMP GetCommandString(UINT_PTR idCmd, UINT uType, UINT *pReserved, CHAR *pszName, UINT cchMax) {
        return E_NOTIMPL;
    }

private:
    HRESULT RunStrataFS(const std::vector<std::wstring>& args, bool showOutput = true) {
        std::wstring cmdLine = L"\"C:\\Program Files\\StrataFS\\stratafs.exe\"";
        for (const auto& arg : args) {
            cmdLine += L" \"" + arg + L"\"";
        }

        STARTUPINFOW si = { sizeof(si) };
        PROCESS_INFORMATION pi;

        if (showOutput) {
            si.dwFlags = STARTF_USESHOWWINDOW;
            si.wShowWindow = SW_SHOW;
        } else {
            si.dwFlags = STARTF_USESHOWWINDOW;
            si.wShowWindow = SW_HIDE;
        }

        if (!CreateProcessW(nullptr, &cmdLine[0], nullptr, nullptr, FALSE, 0, nullptr, nullptr, &si, &pi)) {
            return HRESULT_FROM_WIN32(GetLastError());
        }

        if (showOutput) {
            WaitForSingleObject(pi.hProcess, INFINITE);
        }

        CloseHandle(pi.hProcess);
        CloseHandle(pi.hThread);

        return S_OK;
    }

    HRESULT ViewMetadata() {
        if (m_selectedFiles.empty()) return E_FAIL;
        return RunStrataFS({ L"file", L"info", m_selectedFiles[0] });
    }

    HRESULT ViewChunks() {
        if (m_selectedFiles.empty()) return E_FAIL;
        return RunStrataFS({ L"file", L"chunks", m_selectedFiles[0] });
    }

    HRESULT FindSimilar() {
        if (m_selectedFiles.empty()) return E_FAIL;
        std::wstring url = L"http://localhost:8080/docs?similar=" + m_selectedFiles[0];
        ShellExecuteW(nullptr, L"open", url.c_str(), nullptr, nullptr, SW_SHOW);
        return S_OK;
    }

    HRESULT Reindex() {
        for (const auto& file : m_selectedFiles) {
            RunStrataFS({ L"file", L"reindex", file }, false);
        }

        std::wstring msg = L"Queued " + std::to_wstring(m_selectedFiles.size()) + L" file(s) for reindexing";
        MessageBoxW(nullptr, msg.c_str(), L"StrataFS", MB_OK | MB_ICONINFORMATION);
        return S_OK;
    }

    HRESULT AddSource() {
        if (m_selectedFiles.empty()) return E_FAIL;
        return RunStrataFS({ L"source", L"add", L"--path", m_selectedFiles[0] });
    }

    HRESULT ExportMetadata() {
        if (m_selectedFiles.empty()) return E_FAIL;
        return RunStrataFS({ L"fs", L"export", L"--output", m_selectedFiles[0] });
    }
};

// Class factory
class StrataFSContextMenuFactory : public IClassFactory {
private:
    LONG m_refCount;

public:
    StrataFSContextMenuFactory() : m_refCount(1) {}

    STDMETHODIMP QueryInterface(REFIID riid, void **ppv) {
        if (riid == IID_IUnknown || riid == IID_IClassFactory) {
            *ppv = static_cast<IClassFactory*>(this);
            AddRef();
            return S_OK;
        }
        *ppv = nullptr;
        return E_NOINTERFACE;
    }

    STDMETHODIMP_(ULONG) AddRef() { return InterlockedIncrement(&m_refCount); }
    STDMETHODIMP_(ULONG) Release() {
        LONG ref = InterlockedDecrement(&m_refCount);
        if (ref == 0) delete this;
        return ref;
    }

    STDMETHODIMP CreateInstance(IUnknown *pUnkOuter, REFIID riid, void **ppv) {
        if (pUnkOuter) return CLASS_E_NOAGGREGATION;
        StrataFSContextMenu *menu = new StrataFSContextMenu();
        HRESULT hr = menu->QueryInterface(riid, ppv);
        menu->Release();
        return hr;
    }

    STDMETHODIMP LockServer(BOOL fLock) { return S_OK; }
};

// DLL exports
STDAPI DllGetClassObject(REFCLSID rclsid, REFIID riid, void **ppv) {
    if (rclsid == CLSID_StrataFSContextMenu) {
        StrataFSContextMenuFactory *factory = new StrataFSContextMenuFactory();
        HRESULT hr = factory->QueryInterface(riid, ppv);
        factory->Release();
        return hr;
    }
    *ppv = nullptr;
    return CLASS_E_CLASSNOTAVAILABLE;
}

STDAPI DllCanUnloadNow() { return S_OK; }
STDAPI DllRegisterServer() { return S_OK; }
STDAPI DllUnregisterServer() { return S_OK; }

BOOL APIENTRY DllMain(HMODULE hModule, DWORD reason, LPVOID lpReserved) {
    return TRUE;
}
