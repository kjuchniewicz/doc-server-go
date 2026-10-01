@echo off
rem ============================================================
rem Instalacja autostartu doc-server-go przez Harmonogram zadan.
rem Uruchom jako ADMINISTRATOR (prawy klik -> Uruchom jako administrator).
rem Skrypt sam podmienia sciezke w zadanie.xml na biezacy folder.
rem ============================================================
cd /d "%~dp0"

powershell -Command "(Get-Content '%~dp0zadanie.xml' -Raw) -replace [regex]::Escape('D:\Projekty\ZHU\MichelinDTR\doc-server-go'), ('%~dp0' -replace '\\$','') | Set-Content '%~dp0zadanie.xml' -Encoding Unicode"

schtasks /Create /TN "DocServerGo" /XML "%~dp0zadanie.xml" /F
if errorlevel 1 (
    echo.
    echo BLAD: prawdopodobnie brak uprawnien administratora.
    pause
    exit /b 1
)
echo.
echo Zainstalowano. Serwer wystartuje automatycznie przy starcie systemu.
echo.
echo Polecenia:
echo   schtasks /Run /TN "DocServerGo"        - uruchom teraz
echo   schtasks /End /TN "DocServerGo"        - zatrzymaj
echo   schtasks /Delete /TN "DocServerGo" /F  - odinstaluj
echo.
pause
