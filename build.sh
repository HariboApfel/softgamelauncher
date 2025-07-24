#!/bin/bash

# Game Launcher - Universal Build Script (Unix/Linux/macOS)
# Consolidates all build functionality into one script

set -e

# Default values
TARGET="gui"
TEST=false
RUN=false
CLEAN=false
HELP=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

show_help() {
    echo -e "${CYAN}Game Launcher Build Script${NC}"
    echo
    echo "Usage: ./build.sh [options]"
    echo
    echo "Options:"
    echo "  -t, --target TARGET   What to build: gui, console, cli, or all (default: gui)"
    echo "  -T, --test           Run tests after building"
    echo "  -r, --run            Run the application after building"
    echo "  -c, --clean          Clean build artifacts before building"
    echo "  -h, --help           Show this help message"
    echo
    echo "Examples:"
    echo "  ./build.sh                      # Build GUI version"
    echo "  ./build.sh -t console           # Build console version"
    echo "  ./build.sh -t all -T            # Build all versions and test"
    echo "  ./build.sh -c -r                # Clean, build GUI, and run"
}

check_prerequisites() {
    echo -e "${GREEN}Checking prerequisites...${NC}"
    
    # Check Go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
        echo -e "${YELLOW}Please install Go from https://golang.org/dl/${NC}"
        return 1
    fi
    
    local go_version=$(go version)
    echo -e "${GREEN}✓ Go found: $go_version${NC}"
    
    return 0
}

install_dependencies() {
    echo -e "${GREEN}Installing dependencies...${NC}"
    if ! go mod tidy; then
        echo -e "${RED}Error: Failed to install dependencies${NC}"
        return 1
    fi
    echo -e "${GREEN}✓ Dependencies installed${NC}"
    return 0
}

build_gui() {
    echo -e "${GREEN}Building GUI version...${NC}"
    
    if go build -o gamelauncher main.go; then
        echo -e "${GREEN}✓ GUI build successful${NC}"
        return 0
    else
        echo -e "${RED}✗ GUI build failed${NC}"
        return 1
    fi
}

build_console() {
    echo -e "${GREEN}Building console version...${NC}"
    
    if go build -o gamelauncher_console main_console.go; then
        echo -e "${GREEN}✓ Console build successful${NC}"
        return 0
    else
        echo -e "${RED}✗ Console build failed${NC}"
        return 1
    fi
}

build_cli() {
    echo -e "${GREEN}Building CLI version...${NC}"
    
    # Use console version as CLI version
    if go build -o gamelauncher_cli main_console.go; then
        echo -e "${GREEN}✓ CLI build successful${NC}"
        return 0
    else
        echo -e "${RED}✗ CLI build failed${NC}"
        return 1
    fi
}

clean_artifacts() {
    echo -e "${GREEN}Cleaning build artifacts...${NC}"
    local artifacts=("gamelauncher" "gamelauncher_console" "gamelauncher_cli" "test_paths" "fix_paths")
    
    for artifact in "${artifacts[@]}"; do
        if [ -f "$artifact" ]; then
            rm -f "$artifact"
            echo -e "${YELLOW}Removed $artifact${NC}"
        fi
    done
    
    echo -e "${GREEN}✓ Clean complete${NC}"
}

run_tests() {
    echo -e "${GREEN}Running tests...${NC}"
    
    # Build and run path tests
    echo -e "${YELLOW}Building path tests...${NC}"
    if go build -o test_paths test_paths.go; then
        echo -e "${YELLOW}Running path tests...${NC}"
        if ./test_paths; then
            echo -e "${GREEN}✓ Path tests passed${NC}"
        else
            echo -e "${RED}✗ Path tests failed${NC}"
        fi
    fi
    
    # Run Go tests
    echo -e "${YELLOW}Running Go tests...${NC}"
    if go test ./test_basic.go; then
        echo -e "${GREEN}✓ Go tests passed${NC}"
    else
        echo -e "${RED}✗ Go tests failed${NC}"
    fi
}

run_application() {
    local target_type=$1
    echo -e "${GREEN}Starting application...${NC}"
    
    case $target_type in
        "gui")
            if [ -f "gamelauncher" ]; then
                ./gamelauncher
            fi
            ;;
        "console")
            if [ -f "gamelauncher_console" ]; then
                ./gamelauncher_console
            fi
            ;;
        "cli")
            if [ -f "gamelauncher_cli" ]; then
                ./gamelauncher_cli -help
            fi
            ;;
    esac
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--target)
            TARGET="$2"
            shift 2
            ;;
        -T|--test)
            TEST=true
            shift
            ;;
        -r|--run)
            RUN=true
            shift
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -h|--help)
            HELP=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# Main execution
main() {
    if [ "$HELP" = true ]; then
        show_help
        return 0
    fi
    
    echo -e "${CYAN}Game Launcher Build Script${NC}"
    echo -e "${CYAN}=========================${NC}"
    
    if ! check_prerequisites; then
        exit 1
    fi
    
    if [ "$CLEAN" = true ]; then
        clean_artifacts
    fi
    
    if ! install_dependencies; then
        exit 1
    fi
    
    local success=false
    
    case "${TARGET,,}" in
        "gui")
            if build_gui; then
                success=true
            fi
            ;;
        "console")
            if build_console; then
                success=true
            fi
            ;;
        "cli")
            if build_cli; then
                success=true
            fi
            ;;
        "all")
            gui_success=false
            console_success=false
            cli_success=false
            
            if build_gui; then
                gui_success=true
                success=true
            fi
            
            if build_console; then
                console_success=true
                success=true
            fi
            
            if build_cli; then
                cli_success=true
                success=true
            fi
            
            echo -e "\n${CYAN}Build Summary:${NC}"
            echo -e "GUI:     $([ "$gui_success" = true ] && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}")"
            echo -e "Console: $([ "$console_success" = true ] && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}")"
            echo -e "CLI:     $([ "$cli_success" = true ] && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}")"
            ;;
        *)
            echo -e "${RED}Invalid target: $TARGET${NC}"
            show_help
            exit 1
            ;;
    esac
    
    if [ "$TEST" = true ] && [ "$success" = true ]; then
        run_tests
    fi
    
    if [ "$RUN" = true ] && [ "$success" = true ]; then
        run_application "$TARGET"
    fi
    
    if [ "$success" = true ]; then
        echo -e "\n${GREEN}✓ Build completed successfully!${NC}"
        echo -e "${CYAN}Run ./gamelauncher to start the application.${NC}"
    else
        echo -e "\n${RED}✗ Build failed!${NC}"
        exit 1
    fi
}

# Execute main function
main 