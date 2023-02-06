#!/bin/bash
set -e

SRCDIR=`pwd`
BUILDDIR=`pwd`/build

mkdir -p ${BUILDDIR} 2>/dev/null
cd ${BUILDDIR}
echo "Cloning coredns repo..."
git clone https://github.com/coredns/coredns.git

cd coredns
git checkout v1.8.3

echo "Patching plugin config..."
ed plugin.cfg <<EOED
/rewrite:rewrite
a
pns:github.com/pulsedomains/coredns-pns
.
w
q
EOED

# Add our module to coredns.
echo "Patching go modules..."
ed go.mod <<EOED
a
replace github.com/pulsedomains/coredns-pns => ../..
.
/^)
-1
a
	github.com/pulsedomains/coredns-pns v1.3.3
.
w
q
EOED

go get github.com/pulsedomains/coredns-pns@v1.3.3
go get
go mod download

echo "Building..."
make SHELL='sh -x' CGO_ENABLED=1 coredns

cp coredns ${SRCDIR}
chmod -R 755 .git
cd ${SRCDIR}
rm -r ${BUILDDIR}
