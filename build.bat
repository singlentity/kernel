@echo off
cd /d "%~dp0kernel-app"
echo Собираю KERNEL.exe...
go build -ldflags="-H windowsgui -s -w" -o KERNEL.exe .
if errorlevel 1 ( echo. & echo ОШИБКА СБОРКИ. Установлен ли Go? ^(go version^) & pause & exit /b 1 )
echo.
echo Готово: %~dp0kernel-app\KERNEL.exe
pause
