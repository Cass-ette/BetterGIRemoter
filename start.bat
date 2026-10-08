@echo off
REM BetterGIRemoter 启动脚本
REM 用法: start.bat [BetterGI的API_Token]
REM 不传参数时使用默认 Token

set BETTERGI_URL=http://127.0.0.1:8080
set BETTERGI_TOKEN=5b33465d1b1a45dbb1d68df9653362a5
if not "%~1"=="" set BETTERGI_TOKEN=%~1

REM BetterGI 程序路径：BetterGI 未运行时 Remoter 会自动拉起它
set BETTERGI_EXE=C:\Users\ASUS\better-genshin-impact\BetterGenshinImpact\bin\x64\Release\net8.0-windows10.0.22621.0\BetterGI.exe

set PORT=8000
bettergi-remoter.exe
