@echo off
REM ═══════════════════════════════════════════════════════════════
REM bwenv — CLI wrapper for Windows
REM
REM This script forwards all arguments to bwenv.exe in the same
REM directory. It allows Windows users to type "bwenv" instead of
REM "bwenv.exe" in cmd and PowerShell.
REM ═══════════════════════════════════════════════════════════════
"%~dp0bwenv.exe" %*
