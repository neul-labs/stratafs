/*
 * StrataFS Windows Search IFilter
 *
 * This IFilter implementation exposes StrataFS indexed content to Windows Search.
 * It reads chunk data from the StrataFS database and provides it to the indexer.
 *
 * Build with Visual Studio or cl.exe
 * Register with: regsvr32 StrataFSFilter.dll
 */

#include <windows.h>
#include <filter.h>
#include <filterr.h>
#include <propkey.h>
#include <propvarutil.h>
#include <shlwapi.h>
#include <strsafe.h>
#include <sqlite3.h>
#include <string>
#include <vector>

// {A1B2C3D4-E5F6-7890-ABCD-EF1234567890}
static const GUID CLSID_StrataFSFilter =
{ 0xa1b2c3d4, 0xe5f6, 0x7890, { 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90 } };

class StrataFSFilter : public IFilter, public IPersistFile, public IPersistStream {
private:
    LONG m_refCount;
    std::wstring m_filePath;
    std::vector<std::string> m_chunks;
    size_t m_currentChunk;
    size_t m_currentPos;
    bool m_initialized;

public:
    StrataFSFilter() : m_refCount(1), m_currentChunk(0), m_currentPos(0), m_initialized(false) {}
    virtual ~StrataFSFilter() {}

    // IUnknown
    STDMETHODIMP QueryInterface(REFIID riid, void **ppv) {
        if (riid == IID_IUnknown || riid == IID_IFilter) {
            *ppv = static_cast<IFilter*>(this);
        } else if (riid == IID_IPersistFile) {
            *ppv = static_cast<IPersistFile*>(this);
        } else if (riid == IID_IPersistStream) {
            *ppv = static_cast<IPersistStream*>(this);
        } else {
            *ppv = nullptr;
            return E_NOINTERFACE;
        }
        AddRef();
        return S_OK;
    }

    STDMETHODIMP_(ULONG) AddRef() {
        return InterlockedIncrement(&m_refCount);
    }

    STDMETHODIMP_(ULONG) Release() {
        LONG ref = InterlockedDecrement(&m_refCount);
        if (ref == 0) {
            delete this;
        }
        return ref;
    }

    // IPersist
    STDMETHODIMP GetClassID(CLSID *pClassID) {
        *pClassID = CLSID_StrataFSFilter;
        return S_OK;
    }

    // IPersistFile
    STDMETHODIMP IsDirty() { return S_FALSE; }

    STDMETHODIMP Load(LPCOLESTR pszFileName, DWORD dwMode) {
        m_filePath = pszFileName;
        return LoadFromDatabase();
    }

    STDMETHODIMP Save(LPCOLESTR pszFileName, BOOL fRemember) { return E_NOTIMPL; }
    STDMETHODIMP SaveCompleted(LPCOLESTR pszFileName) { return E_NOTIMPL; }
    STDMETHODIMP GetCurFile(LPOLESTR *ppszFileName) { return E_NOTIMPL; }

    // IPersistStream
    STDMETHODIMP Load(IStream *pStm) { return E_NOTIMPL; }
    STDMETHODIMP Save(IStream *pStm, BOOL fClearDirty) { return E_NOTIMPL; }
    STDMETHODIMP GetSizeMax(ULARGE_INTEGER *pcbSize) { return E_NOTIMPL; }

    // IFilter
    STDMETHODIMP Init(ULONG grfFlags, ULONG cAttributes, const FULLPROPSPEC *aAttributes, ULONG *pFlags) {
        *pFlags = IFILTER_FLAGS_OLE_PROPERTIES;
        m_currentChunk = 0;
        m_currentPos = 0;
        m_initialized = true;
        return S_OK;
    }

    STDMETHODIMP GetChunk(STAT_CHUNK *pStat) {
        if (!m_initialized || m_currentChunk >= m_chunks.size()) {
            return FILTER_E_END_OF_CHUNKS;
        }

        pStat->idChunk = static_cast<ULONG>(m_currentChunk + 1);
        pStat->breakType = CHUNK_EOS;
        pStat->flags = CHUNK_TEXT;
        pStat->locale = GetUserDefaultLCID();
        pStat->attribute.guidPropSet = PSGUID_STORAGE;
        pStat->attribute.psProperty.ulKind = PRSPEC_PROPID;
        pStat->attribute.psProperty.propid = PID_STG_CONTENTS;
        pStat->idChunkSource = pStat->idChunk;
        pStat->cwcStartSource = 0;
        pStat->cwcLenSource = 0;

        m_currentPos = 0;
        return S_OK;
    }

    STDMETHODIMP GetText(ULONG *pcwcBuffer, WCHAR *awcBuffer) {
        if (!m_initialized || m_currentChunk >= m_chunks.size()) {
            return FILTER_E_NO_MORE_TEXT;
        }

        const std::string& chunk = m_chunks[m_currentChunk];

        if (m_currentPos >= chunk.length()) {
            m_currentChunk++;
            return FILTER_E_NO_MORE_TEXT;
        }

        // Convert UTF-8 to wide string
        size_t remaining = chunk.length() - m_currentPos;
        size_t toCopy = min(remaining, static_cast<size_t>(*pcwcBuffer - 1));

        int converted = MultiByteToWideChar(
            CP_UTF8, 0,
            chunk.c_str() + m_currentPos, static_cast<int>(toCopy),
            awcBuffer, static_cast<int>(*pcwcBuffer - 1)
        );

        if (converted > 0) {
            awcBuffer[converted] = L'\0';
            *pcwcBuffer = converted;
            m_currentPos += toCopy;
        } else {
            *pcwcBuffer = 0;
        }

        return (m_currentPos >= chunk.length()) ? FILTER_S_LAST_TEXT : S_OK;
    }

    STDMETHODIMP GetValue(PROPVARIANT **ppPropValue) {
        return FILTER_E_NO_MORE_VALUES;
    }

    STDMETHODIMP BindRegion(FILTERREGION origPos, REFIID riid, void **ppunk) {
        return E_NOTIMPL;
    }

private:
    HRESULT LoadFromDatabase() {
        m_chunks.clear();

        // Get StrataFS database path
        wchar_t userProfile[MAX_PATH];
        if (!GetEnvironmentVariableW(L"USERPROFILE", userProfile, MAX_PATH)) {
            return E_FAIL;
        }

        std::wstring dbPath = userProfile;
        dbPath += L"\\.stratafs\\stratafs.db";

        // Convert to UTF-8 for SQLite
        char dbPathUtf8[MAX_PATH * 3];
        WideCharToMultiByte(CP_UTF8, 0, dbPath.c_str(), -1, dbPathUtf8, sizeof(dbPathUtf8), nullptr, nullptr);

        char filePathUtf8[MAX_PATH * 3];
        WideCharToMultiByte(CP_UTF8, 0, m_filePath.c_str(), -1, filePathUtf8, sizeof(filePathUtf8), nullptr, nullptr);

        sqlite3 *db;
        if (sqlite3_open(dbPathUtf8, &db) != SQLITE_OK) {
            return E_FAIL;
        }

        // Query for file ID
        const char *fileSql = "SELECT id FROM files WHERE path = ? AND deleted_at IS NULL";
        sqlite3_stmt *stmt;

        if (sqlite3_prepare_v2(db, fileSql, -1, &stmt, nullptr) != SQLITE_OK) {
            sqlite3_close(db);
            return E_FAIL;
        }

        sqlite3_bind_text(stmt, 1, filePathUtf8, -1, SQLITE_TRANSIENT);

        if (sqlite3_step(stmt) != SQLITE_ROW) {
            sqlite3_finalize(stmt);
            sqlite3_close(db);
            return E_FAIL;
        }

        int64_t fileId = sqlite3_column_int64(stmt, 0);
        sqlite3_finalize(stmt);

        // Get chunks
        const char *chunkSql = "SELECT content FROM file_chunks WHERE file_id = ? ORDER BY chunk_index";
        if (sqlite3_prepare_v2(db, chunkSql, -1, &stmt, nullptr) == SQLITE_OK) {
            sqlite3_bind_int64(stmt, 1, fileId);

            while (sqlite3_step(stmt) == SQLITE_ROW) {
                const char *content = reinterpret_cast<const char*>(sqlite3_column_text(stmt, 0));
                if (content) {
                    m_chunks.push_back(content);
                }
            }
            sqlite3_finalize(stmt);
        }

        sqlite3_close(db);
        return m_chunks.empty() ? E_FAIL : S_OK;
    }
};

// Class factory
class StrataFSFilterFactory : public IClassFactory {
private:
    LONG m_refCount;

public:
    StrataFSFilterFactory() : m_refCount(1) {}

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
        StrataFSFilter *filter = new StrataFSFilter();
        HRESULT hr = filter->QueryInterface(riid, ppv);
        filter->Release();
        return hr;
    }

    STDMETHODIMP LockServer(BOOL fLock) { return S_OK; }
};

// DLL exports
static LONG g_serverLocks = 0;

STDAPI DllGetClassObject(REFCLSID rclsid, REFIID riid, void **ppv) {
    if (rclsid == CLSID_StrataFSFilter) {
        StrataFSFilterFactory *factory = new StrataFSFilterFactory();
        HRESULT hr = factory->QueryInterface(riid, ppv);
        factory->Release();
        return hr;
    }
    *ppv = nullptr;
    return CLASS_E_CLASSNOTAVAILABLE;
}

STDAPI DllCanUnloadNow() {
    return (g_serverLocks == 0) ? S_OK : S_FALSE;
}

STDAPI DllRegisterServer() {
    // Registration would set up registry entries for the IFilter
    // This is typically done via a .reg file or installer
    return S_OK;
}

STDAPI DllUnregisterServer() {
    return S_OK;
}

BOOL APIENTRY DllMain(HMODULE hModule, DWORD reason, LPVOID lpReserved) {
    switch (reason) {
        case DLL_PROCESS_ATTACH:
            DisableThreadLibraryCalls(hModule);
            break;
    }
    return TRUE;
}
