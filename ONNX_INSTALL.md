# Installing ONNX Runtime for AgentFS

If you want to use the full embedding capabilities of AgentFS instead of the mock implementation, you'll need to install the ONNX Runtime library.

## For macOS (Apple Silicon)

1. Download the ONNX Runtime for macOS ARM64:
   ```bash
   curl -L -O https://github.com/microsoft/onnxruntime/releases/download/v1.16.0/onnxruntime-osx-arm64-1.16.0.tgz
   ```

2. Extract the archive:
   ```bash
   tar -xzf onnxruntime-osx-arm64-1.16.0.tgz
   ```

3. Copy the library to a standard location:
   ```bash
   sudo cp onnxruntime-osx-arm64-1.16.0/lib/libonnxruntime.1.16.0.dylib /usr/local/lib/
   sudo ln -s /usr/local/lib/libonnxruntime.1.16.0.dylib /usr/local/lib/onnxruntime.so
   ```

## For macOS (Intel)

1. Download the ONNX Runtime for macOS x64:
   ```bash
   curl -L -O https://github.com/microsoft/onnxruntime/releases/download/v1.16.0/onnxruntime-osx-x64-1.16.0.tgz
   ```

2. Extract the archive:
   ```bash
   tar -xzf onnxruntime-osx-x64-1.16.0.tgz
   ```

3. Copy the library to a standard location:
   ```bash
   sudo cp onnxruntime-osx-x64-1.16.0/lib/libonnxruntime.1.16.0.dylib /usr/local/lib/
   sudo ln -s /usr/local/lib/libonnxruntime.1.16.0.dylib /usr/local/lib/onnxruntime.so
   ```

## For Linux

1. Download the ONNX Runtime for Linux:
   ```bash
   curl -L -O https://github.com/microsoft/onnxruntime/releases/download/v1.16.0/onnxruntime-linux-x64-1.16.0.tgz
   ```

2. Extract the archive:
   ```bash
   tar -xzf onnxruntime-linux-x64-1.16.0.tgz
   ```

3. Copy the library to a standard location:
   ```bash
   sudo cp onnxruntime-linux-x64-1.16.0/lib/libonnxruntime.so.1.16.0 /usr/local/lib/
   sudo ln -s /usr/local/lib/libonnxruntime.so.1.16.0 /usr/local/lib/onnxruntime.so
   ```

## For Windows

1. Download the ONNX Runtime for Windows:
   - Visit https://github.com/microsoft/onnxruntime/releases
   - Download the appropriate version for your system

2. Extract the archive and add the library to your PATH

After installing the ONNX Runtime, restart your terminal and run AgentFS again. It should now use the real embedding model instead of the mock implementation.