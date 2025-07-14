#!/bin/bash

# RTTYS Ubuntu Package Build Script

set -e

# Configuration
PACKAGE_NAME="rttys"
VERSION=$(grep 'const RttysVersion' main.go | cut -d'"' -f2 | sed 's/^v//')
MAINTAINER="Jianhui Zhao <zhaojh329@gmail.com>"
DESCRIPTION="Access your device's terminal from anywhere via the web"
URL="https://github.com/zhaojh329/rttys"
GitCommit=$(git log --pretty=format:"%h" -1)
BuildTime=$(date +%FT%T%z)

ARCH="$1"  # Pass architecture as an argument, e.g., amd64 or arm64

[ -z "$ARCH" ] && {
    echo "Usage: $0 <arch>";
    echo "Example: $0 amd64"
    exit 1;
}

# Build directory
BUILD_DIR="build-deb"
INSTALL_DIR="$BUILD_DIR/usr"

echo "Building RTTYS v$VERSION for Ubuntu..."

# Clean and create build directory
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR/{usr/bin,etc/rttys,lib/systemd/system}

# Build the binary
echo "Building binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=$ARCH go build -ldflags "-s -w -X main.GitCommit=$GitCommit -X main.BuildTime=$BuildTime" -o $INSTALL_DIR/bin/rttys .

# Copy configuration files
echo "Copying configuration files..."
cp rttys.conf $BUILD_DIR/etc/rttys/
cp rttys.service $BUILD_DIR/lib/systemd/system/

# Create postinstall script
cat > $BUILD_DIR/postinstall.sh << 'EOF'
#!/bin/bash

# Enable and start the service
systemctl daemon-reload
systemctl enable rttys
echo "RTTYS installed successfully!"
echo "Configuration file: /etc/rttys/rttys.conf"
echo "Start service: sudo systemctl start rttys"
echo "View logs: sudo journalctl -u rttys -f"
EOF

# Create preremove script
cat > $BUILD_DIR/preremove.sh << 'EOF'
#!/bin/bash
# Stop and disable the service
systemctl stop rttys || true
systemctl disable rttys || true
EOF

# Create postremove script
cat > $BUILD_DIR/postremove.sh << 'EOF'
#!/bin/bash
EOF

# Build the package
echo "Creating .deb package..."
fpm -s dir -t deb \
    --name "$PACKAGE_NAME" \
    --version "$VERSION" \
    --maintainer "$MAINTAINER" \
    --description "$DESCRIPTION" \
    --url "$URL" \
    --architecture "$ARCH" \
    --depends "systemd" \
    --deb-no-default-config-files \
    --config-files "/etc/rttys/rttys.conf" \
    --after-install "$BUILD_DIR/postinstall.sh" \
    --before-remove "$BUILD_DIR/preremove.sh" \
    --after-remove "$BUILD_DIR/postremove.sh" \
    -C "$BUILD_DIR" \
    .

echo "Package created: ${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
echo "Install with: sudo dpkg -i ${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"
