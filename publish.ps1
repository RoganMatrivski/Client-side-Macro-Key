./build_release.ps1
./tools/upx.exe --best --ultra-brute -v ./build/release/main.exe
Compress-Archive -Path "./build/release/main.exe","./LICENSE","./about.txt","./configuration.json" -DestinationPath MacroKeyApp -Force
