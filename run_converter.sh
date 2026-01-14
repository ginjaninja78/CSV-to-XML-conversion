#!/bin/bash
# =============================================================================
# CSV to XML Converter - Shell Runner
# =============================================================================
#
# This shell script provides a simple way to run the CSV to XML converter
# on Linux and macOS systems.
#
# USAGE:
#   1. Place your CSV files in the "input" folder.
#   2. Run: ./run_converter.sh
#   3. Check the "output" folder for the generated XML files.
#
# CUSTOMIZATION:
#   - Modify the DEPARTMENT variable to change the default department.
#   - Add command-line arguments for different processing modes.
#
# =============================================================================

# Exit on error.
set -e

# Display header.
echo ""
echo "================================================================================"
echo "                         CSV to XML Converter"
echo "================================================================================"
echo ""

# Get the directory where this script is located.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Change to the script directory.
cd "$SCRIPT_DIR"

# Determine the executable name based on the OS.
if [[ "$OSTYPE" == "darwin"* ]]; then
    EXECUTABLE="csv2xml-darwin"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    EXECUTABLE="csv2xml-linux"
else
    EXECUTABLE="csv2xml"
fi

# Check if the converter executable exists.
if [ ! -f "$EXECUTABLE" ]; then
    # Try the generic name.
    if [ -f "csv2xml" ]; then
        EXECUTABLE="csv2xml"
    else
        echo "ERROR: Converter executable not found!"
        echo ""
        echo "Please ensure the converter executable is in the same directory as this script."
        echo "Expected: $EXECUTABLE or csv2xml"
        echo ""
        exit 1
    fi
fi

# Ensure the executable has execute permissions.
chmod +x "$EXECUTABLE"

# Create directories if they don't exist.
mkdir -p input output input_archive output_archive

# Count the number of CSV files in the input directory.
FILE_COUNT=$(find input -maxdepth 1 -name "*.csv" -type f 2>/dev/null | wc -l)

if [ "$FILE_COUNT" -eq 0 ]; then
    echo ""
    echo "No CSV files found in the input folder."
    echo ""
    echo "Please place your CSV files in the 'input' folder and run this script again."
    echo ""
    exit 0
fi

echo "Found $FILE_COUNT CSV file(s) in the input folder."
echo ""

# =============================================================================
# DEPARTMENT SELECTION
# =============================================================================
#
# OPTION 1: Hardcode the department (uncomment and modify the line below).
# DEPARTMENT="claims"
#
# OPTION 2: Use command-line argument.
# DEPARTMENT="${1:-claims}"
#
# OPTION 3: Interactive selection (uncomment the section below).
#
# echo "Available departments:"
# echo "  1. claims"
# echo "  2. underwriting"
# echo "  3. accounting"
# echo "  4. operations"
# echo ""
# read -p "Select department (1-4): " DEPT_CHOICE
#
# case $DEPT_CHOICE in
#     1) DEPARTMENT="claims" ;;
#     2) DEPARTMENT="underwriting" ;;
#     3) DEPARTMENT="accounting" ;;
#     4) DEPARTMENT="operations" ;;
#     *) DEPARTMENT="claims" ;;
# esac

# =============================================================================
# RUN THE CONVERTER
# =============================================================================

echo "Starting conversion..."
echo ""

# Run the converter with the process command.
#
# COMMAND OPTIONS:
#   process           - Process all CSV files in the input directory.
#   --department, -d  - Specify the department code.
#   --type, -t        - Specify the transaction type.
#   --config, -c      - Specify a custom configuration file.
#   --verbose, -v     - Enable verbose output.
#   --dry-run         - Validate without generating output.
#
# CUSTOMIZATION:
#   Modify the command below to add or remove options as needed.

./"$EXECUTABLE" process --verbose

# Check the exit code.
if [ $? -eq 0 ]; then
    echo ""
    echo "================================================================================"
    echo "                         Conversion Complete!"
    echo "================================================================================"
    echo ""
    echo "Check the 'output' folder for the generated XML files."
    echo "Processed CSV files have been moved to 'input_archive'."
    echo ""
else
    echo ""
    echo "================================================================================"
    echo "                         Conversion Failed!"
    echo "================================================================================"
    echo ""
    echo "Please check the error messages above and the error log in the output folder."
    echo ""
    exit 1
fi
