@echo off
rem Uruchamia doc-server-go w tle (bez okna konsoli), logi do serwer.log
cd /d "%~dp0"
powershell -WindowStyle Hidden -Command "Start-Process -FilePath '%~dp0doc-server-go.exe' -WorkingDirectory '%~dp0' -WindowStyle Hidden -RedirectStandardOutput '%~dp0serwer.log' -RedirectStandardError '%~dp0serwer-err.log'"
echo Serwer uruchomiony w tle. Logi: serwer.log / serwer-err.log
