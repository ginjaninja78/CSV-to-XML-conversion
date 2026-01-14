@echo off
REM =============================================================================
REM CSV to XML Converter - Batch Runner
REM =============================================================================
REM
REM This batch file provides a simple way to run the CSV to XML converter.
REM Simply double-click this file to process all CSV files in the input folder.
REM
REM USAGE:
REM   1. Place your CSV files in the "input" folder.
REM   2. Double-click this batch file.
REM   3. Check the "output" folder for the generated XML files.
REM   4. Check the console output for any errors.
REM
REM CUSTOMIZATION:
REM   - Modify the DEPARTMENT variable to change the default department.
REM   - Add command-line arguments for different processing modes.
REM
REM =============================================================================

REM Set the title of the console window.
title CSV to XML Converter

REM Display header.
echo.
echo ================================================================================
echo                         CSV to XML Converter
echo ================================================================================
echo.

REM Get the directory where this batch file is located.
set "SCRIPT_DIR=%~dp0"

REM Change to the script directory.
cd /d "%SCRIPT_DIR%"

REM Check if the converter executable exists.
if not exist "csv2xml.exe" (
    echo ERROR: csv2xml.exe not found!
    echo.
    echo Please ensure the converter executable is in the same directory as this batch file.
    echo.
    pause
    exit /b 1
)

REM Check if the input directory exists.
if not exist "input" (
    echo Creating input directory...
    mkdir input
)

REM Check if the output directory exists.
if not exist "output" (
    echo Creating output directory...
    mkdir output
)

REM Check if the input_archive directory exists.
if not exist "input_archive" (
    echo Creating input_archive directory...
    mkdir input_archive
)

REM Check if the output_archive directory exists.
if not exist "output_archive" (
    echo Creating output_archive directory...
    mkdir output_archive
)

REM Count the number of CSV files in the input directory.
set "FILE_COUNT=0"
for %%f in (input\*.csv) do set /a FILE_COUNT+=1

if %FILE_COUNT%==0 (
    echo.
    echo No CSV files found in the input folder.
    echo.
    echo Please place your CSV files in the "input" folder and run this script again.
    echo.
    pause
    exit /b 0
)

echo Found %FILE_COUNT% CSV file(s) in the input folder.
echo.

REM =============================================================================
REM DEPARTMENT SELECTION
REM =============================================================================
REM
REM OPTION 1: Hardcode the department (uncomment and modify the line below).
REM set "DEPARTMENT=claims"
REM
REM OPTION 2: Prompt the user to select a department.
REM Uncomment the section below to enable interactive department selection.

REM echo Available departments:
REM echo   1. claims
REM echo   2. underwriting
REM echo   3. accounting
REM echo   4. operations
REM echo.
REM set /p DEPT_CHOICE="Select department (1-4): "
REM
REM if "%DEPT_CHOICE%"=="1" set "DEPARTMENT=claims"
REM if "%DEPT_CHOICE%"=="2" set "DEPARTMENT=underwriting"
REM if "%DEPT_CHOICE%"=="3" set "DEPARTMENT=accounting"
REM if "%DEPT_CHOICE%"=="4" set "DEPARTMENT=operations"

REM =============================================================================
REM RUN THE CONVERTER
REM =============================================================================

echo Starting conversion...
echo.

REM Run the converter with the process command.
REM
REM COMMAND OPTIONS:
REM   process           - Process all CSV files in the input directory.
REM   --department, -d  - Specify the department code.
REM   --type, -t        - Specify the transaction type.
REM   --config, -c      - Specify a custom configuration file.
REM   --verbose, -v     - Enable verbose output.
REM   --dry-run         - Validate without generating output.
REM
REM CUSTOMIZATION:
REM   Modify the command below to add or remove options as needed.

csv2xml.exe process --verbose

REM Check the exit code.
if %ERRORLEVEL%==0 (
    echo.
    echo ================================================================================
    echo                         Conversion Complete!
    echo ================================================================================
    echo.
    echo Check the "output" folder for the generated XML files.
    echo Processed CSV files have been moved to "input_archive".
    echo.
) else (
    echo.
    echo ================================================================================
    echo                         Conversion Failed!
    echo ================================================================================
    echo.
    echo Please check the error messages above and the error log in the output folder.
    echo.
)

REM Pause to allow the user to see the output.
echo Press any key to exit...
pause >nul
