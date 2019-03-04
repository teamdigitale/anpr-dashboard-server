Script used to generate the images:

for i in `seq 1 300`; do convert ./mypin-resized.png -fuzz 90% -fill "$(printf '#%06X' $(( $(( 0xFE7569 + $(( i * 11117 * 300 )) )) % 0xFFFFFF )) )" -opaque red ./generated/mypin-`printf %03d $i`.png; done;

