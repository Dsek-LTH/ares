EXECUTABLES = go deno
K := $(foreach exec,$(EXECUTABLES),\
    $(if $(shell which $(exec)),some string,$(error "No '$(exec)' in PATH. Please install $(exec) or edit PATH and try again")))


# TODO: Seems to work, but maybe use air to notify proxy after tailwind has generated css file
# to avoid race condition.
live:
	deno run --allow-run --allow-env --allow-read npm:concurrently --kill-others \
      "templ generate --watch --proxy=http://localhost:8080 --open-browser=false --cmd 'go run .'" \
      "deno run watch" \
      "sleep 1 && air \
    	--build.cmd='templ generate --notify-proxy' \
    	--build.delay=100 \
        --build.bin=true \
    	--build.exclude_dir='' \
        --misc.clean_on_exit \
    	--build.include_dir=assets \
    	--build.include_ext=js,css"

install:
	deno i --allow-scripts
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/air-verse/air@latest
	@echo ""
	@echo "✅ Done!"
	@echo "Remember to add ~/go/bin to you path"
	@echo ""
