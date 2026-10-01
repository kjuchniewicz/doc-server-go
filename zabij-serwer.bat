@echo off
rem Zabija wszystkie dzialajace procesy doc-server-go
echo Zabijanie procesow doc-server-go.exe ...
taskkill /IM doc-server-go.exe /F
if errorlevel 1 (echo Brak dzialajacych procesow serwera.) else (echo Zatrzymano serwer.)
