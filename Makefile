live/server:
	templ generate --watch --proxy="http://localhost:8080" --open-browser=false --cmd="go run ."

# TODO: Run tailwind in watch mode instead and notify proxy with air when css in generated?
live/tailwind:
	go run github.com/cosmtrek/air@v1.51.0 \
	--build.cmd "deno run build && templ generate --notify-proxy" \
	--build.bin "true" \
	--build.delay "100" \
	--build.exclude_dir "" \
	--build.include_dir "components" \
	--build.include_ext "go"

live:
	make -j live/server live/tailwind
