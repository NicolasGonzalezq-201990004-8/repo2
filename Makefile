.PHONY: all proto clean

# Raíz de los .proto
PROTO_ROOT := proto
# Carpetas que contienen cada servicio
PROTO_DIRS := broker common consensus coordinator datanode

# alias (debe coincidir con el alias que usás en option go_package)
# ejemplo: option go_package = "proto/datanode/dpb;dpb"
SUBDIR_broker := bpb
SUBDIR_common := cmpb
SUBDIR_consensus := cspb
SUBDIR_coordinator := cdpb
SUBDIR_datanode := dpb

all: proto

proto:
	@echo "==> Generando .pb.go"
	@for dir in $(PROTO_DIRS); do \
		proto_dir=$(PROTO_ROOT)/$$dir; \
		proto_file=$$proto_dir/$$dir.proto; \
		if [ ! -f $$proto_file ]; then \
			echo "❌ ERROR: $$proto_file no existe"; exit 1; \
		fi; \
		if [ "$$dir" = "broker" ]; then subdir=$(SUBDIR_broker); fi; \
		if [ "$$dir" = "common" ]; then subdir=$(SUBDIR_common); fi; \
		if [ "$$dir" = "consensus" ]; then subdir=$(SUBDIR_consensus); fi; \
		if [ "$$dir" = "coordinator" ]; then subdir=$(SUBDIR_coordinator); fi; \
		if [ "$$dir" = "datanode" ]; then subdir=$(SUBDIR_datanode); fi; \
		out_dir=$$proto_dir/$$subdir; \
		echo " - Compilando $$proto_file -> $$out_dir"; \
		mkdir -p $$out_dir; \
		protoc \
			--go_out=..  \
			--go-grpc_out=..  \
			$$proto_file || { echo "❌ Error compilando $$proto_file"; exit 1; }; \
	done
	@echo "✅ Generación completada correctamente."

clean:
	@echo "==> Limpiando archivos generados..."
	@for dir in $(PROTO_DIRS); do \
		if [ "$$dir" = "broker" ]; then subdir=$(SUBDIR_broker); fi; \
		if [ "$$dir" = "common" ]; then subdir=$(SUBDIR_common); fi; \
		if [ "$$dir" = "consensus" ]; then subdir=$(SUBDIR_consensus); fi; \
		if [ "$$dir" = "coordinator" ]; then subdir=$(SUBDIR_coordinator); fi; \
		if [ "$$dir" = "datanode" ]; then subdir=$(SUBDIR_datanode); fi; \
		rm -rf $(PROTO_ROOT)/$$dir/$$subdir; \
	done
	@echo "🧹 Limpieza completa."
