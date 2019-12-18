# Maintainer: None yet

pkgname=run
pkgver=0.1
pkgrel=1
pkgdesc="Easily manage and invoke small scripts and wrappers"
arch=('i686' 'x86_64')
url="https://github.com/TekWizely/run"
license=('MIT')
makedepends=(
  'go-pie'
  'git'
)
source=("git+https://github.com/TekWizely/run.git")
sha256sums=('SKIP')

pkgver() {
  cd "${srcdir}/run"
  ( set -o pipefail
    git describe --long 2>/dev/null | sed 's/\([^-]*-g\)/r\1/;s/-/./g' ||
    printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
  )
}

build(){
  export GOPATH="${srcdir}/gopath"
  cd "${srcdir}/run"
  go build \
    -trimpath \
    -ldflags "-extldflags $LDFLAGS" 
}

package(){
  cd "${srcdir}/run"
  install -Dm755 run "${pkgdir}/usr/bin/run"
  install -Dm644 LICENSE "${pkgdir}/usr/share/licenses/${pkgname}/LICENSE"
}
