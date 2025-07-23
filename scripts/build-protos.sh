#!/bin/bash

# Script para generar código desde archivos Protocol Buffers
# Este script genera código tanto para Go como para Android/Kotlin

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Directorios
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$PROJECT_ROOT/proto"
GO_OUT_DIR="$PROJECT_ROOT/server-go/proto"
ANDROID_PROTO_DIR="$PROJECT_ROOT/client-android/app/src/main/proto"

echo -e "${GREEN}🔧 Generando código desde archivos Protocol Buffers...${NC}"

# Verificar que protoc esté instalado
if ! command -v protoc &> /dev/null; then
    echo -e "${RED}❌ Error: protoc no está instalado${NC}"
    echo "Instala Protocol Buffers compiler:"
    echo "  macOS: brew install protobuf"
    echo "  Ubuntu/Debian: sudo apt-get install protobuf-compiler"
    echo "  Arch: sudo pacman -S protobuf"
    exit 1
fi

# Verificar que los archivos .proto existan
if [ ! -d "$PROTO_DIR" ] || [ -z "$(ls -A "$PROTO_DIR"/*.proto 2>/dev/null)" ]; then
    echo -e "${RED}❌ Error: No se encontraron archivos .proto en $PROTO_DIR${NC}"
    exit 1
fi

echo -e "${YELLOW} Directorios:${NC}"
echo "  Proto source: $PROTO_DIR"
echo "  Go output: $GO_OUT_DIR"
echo "  Android proto: $ANDROID_PROTO_DIR"

# Crear directorios de salida
mkdir -p "$GO_OUT_DIR"
mkdir -p "$ANDROID_PROTO_DIR"

# Copiar archivos .proto para Android
echo -e "${GREEN} Copiando archivos .proto para Android...${NC}"
cp "$PROTO_DIR"/*.proto "$ANDROID_PROTO_DIR/"

# Generar código Go
echo -e "${GREEN}🐹 Generando código Go...${NC}"

# Verificar que los plugins de Go estén instalados
if ! command -v protoc-gen-go &> /dev/null; then
    echo -e "${YELLOW}⚠️  Instalando protoc-gen-go...${NC}"
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo -e "${YELLOW}⚠️  Instalando protoc-gen-go-grpc...${NC}"
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Generar código Go
protoc --proto_path="$PROTO_DIR" \
    --go_out="$GO_OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$GO_OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    "$PROTO_DIR"/*.proto

echo -e "${GREEN} Código Go generado exitosamente${NC}"

# Verificar archivos generados
echo -e "${YELLOW}📄 Archivos generados:${NC}"

echo -e "${YELLOW}  Go:${NC}"
find "$GO_OUT_DIR" -name "*.pb.go" -o -name "*_grpc.pb.go" | sed 's/^/    /'

echo -e "${YELLOW}  Android:${NC}"
find "$ANDROID_PROTO_DIR" -name "*.proto" | sed 's/^/    /'

echo -e "${GREEN}🎉 ¡Generación completada exitosamente!${NC}"

# Instrucciones adicionales
echo -e "${YELLOW}📝 Próximos pasos:${NC}"
echo "  1. Para Go: El código ya está listo para usar"
echo "  2. Para Android: Ejecuta './gradlew build' en client-android/"
echo "  3. Los archivos .proto en Android se compilarán automáticamente durante el build"

# Si estamos en desarrollo, ofrecer compilar Android
if [ "$1" = "--build-android" ]; then
    echo -e "${GREEN}🤖 Construyendo proyecto Android...${NC}"
    cd "$PROJECT_ROOT/client-android"
    if [ -f "./gradlew" ]; then
        ./gradlew build
        echo -e "${GREEN} Proyecto Android construido exitosamente${NC}"
    else
        echo -e "${YELLOW}⚠️  No se encontró gradlew. Construye manualmente el proyecto Android${NC}"
    fi
fi