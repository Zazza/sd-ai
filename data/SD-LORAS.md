# LoRA Models Reference for Stable Diffusion

## Characters & People

### detail_tweaker
- **Weight:** 0.5–1.0
- **Base:** SD 1.5
- **Description:** Adjusts the level of detail in generated images. Use positive values to enhance detail or negative values to reduce it.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** detail, enhancement, tweak, face, body

### add_detail
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Increases overall image detail and sharpness across the entire composition. Particularly effective on skin textures and fabric.
- **Use with:** Realistic and semi-realistic SD 1.5 checkpoints
- **Tags:** detail, sharpness, texture, quality

### flat_color
- **Weight:** 0.6–1.0
- **Base:** SD 1.5
- **Description:** Flattens colors and removes shading gradients for a clean anime cel-look. Ideal for flat illustration styles.
- **Use with:** Anime SD 1.5 checkpoints (Anything, NAI-derived)
- **Tags:** anime, flat, color, illustration, cel

### epiNoiseoffset_v2
- **Weight:** 0.3–0.7
- **Base:** SD 1.5
- **Description:** Adds noise offset to improve contrast in dark scenes. Produces richer blacks and more realistic lighting in low-light compositions.
- **Use with:** Any SD 1.5 checkpoint, especially realistic ones
- **Tags:** noise, offset, dark, lighting, contrast

### badhandv4
- **Weight:** 0.5–0.8
- **Base:** SD 1.5
- **Description:** Improves hand generation by reducing common artifacts like extra fingers and malformed joints. Best used with negative prompts.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** hand, fix, anatomy, fingers, correction

### corneo_hand_fix
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Trained specifically to correct hand anatomy. Reduces fused fingers and improves hand pose accuracy.
- **Use with:** Realistic and anime SD 1.5 checkpoints
- **Tags:** hand, fix, anatomy, pose

### realistic_hands
- **Weight:** 0.5–0.8
- **Base:** Universal
- **Description:** Produces photorealistic hands with proper proportions, knuckle detail, and natural skin texture. Works well for close-up hand shots.
- **Use with:** Photorealistic checkpoints (SD 1.5 and SDXL)
- **Tags:** hand, realistic, photorealistic, skin, anatomy

### expressive_faces
- **Weight:** 0.4–0.8
- **Base:** SD 1.5
- **Description:** Expands the range of facial expressions beyond the default neutral look. Adds nuance to smiles, frowns, and subtle emotions.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** face, expression, emotion, nuance

### eye_details
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Enhances iris patterns, reflections, eyelash detail, and sclera clarity. Creates striking close-up eye shots.
- **Use with:** Anime and realistic SD 1.5 checkpoints
- **Tags:** eye, iris, detail, reflection, close-up

### hair_detail
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Adds intricate individual hair strands, flyaways, and realistic hair texture. Works on all hair types and lengths.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** hair, strand, detail, texture, flyaway

## Style

### ink_style
- **Weight:** 0.6–1.0
- **Base:** SD 1.5
- **Description:** Traditional ink drawing style with bold strokes, crosshatching, and ink wash effects. Simulates brush and pen ink techniques.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** ink, drawing, traditional, crosshatch, monochrome

### watercolor_v2
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Watercolor painting effect with visible paper texture, color bleeding, and soft wet edges. Mimics traditional watercolor media.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** watercolor, painting, paper, bleed, wet, traditional

### oil_painting
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Oil paint texture with visible brushstrokes, impasto effects, and rich color blending. Gives images a classical painterly quality.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** oil, painting, brushstroke, impasto, classical

### pencil_sketch
- **Weight:** 0.6–1.0
- **Base:** SD 1.5
- **Description:** Graphite pencil drawing style with hatching, shading, and visible pencil marks. Produces monochrome sketch-like output.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** pencil, sketch, graphite, hatching, monochrome

### charcoal_draw
- **Weight:** 0.6–1.0
- **Base:** SD 1.5
- **Description:** Charcoal drawing effect with smudged lines, rich dark values, and textured paper grain. High contrast and expressive marks.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** charcoal, drawing, smudge, texture, dark

### pastel_style
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Soft pastel chalk effect with gentle blending, chalky texture, and muted but warm color palette. Ideal for dreamy compositions.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** pastel, chalk, soft, blend, warm

### marker_render
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Alcohol marker rendering style with visible streaks, vivid saturated colors, and marker bleed-through. Mimics Copics and Prismacolors.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** marker, alcohol, render, vivid, streak

### cel_shading
- **Weight:** 0.5–1.0
- **Base:** SD 1.5
- **Description:** Clean anime cel shading with sharp shadow edges, flat color areas, and minimal gradient. Classic 2D animation look.
- **Use with:** Anime SD 1.5 checkpoints
- **Tags:** anime, cel, shading, flat, animation

### gouache_painting
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Gouache paint texture with opaque matte finish, bold color blocks, and visible brush marks.介于水彩和油画之间的媒介感。
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** gouache, painting, opaque, matte, brush

### crayon_style
- **Weight:** 0.6–1.0
- **Base:** SD 1.5
- **Description:** Crayon and children's art style with waxy texture, imperfect lines, and bright primary colors. Naive and playful aesthetic.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** crayon, children, waxy, naive, playful

### pixel_art_16bit
- **Weight:** 0.6–1.0
- **Base:** SD 1.5
- **Description:** Retro 16-bit pixel art style reminiscent of SNES and Sega Genesis era games. Detailed sprites with limited color palettes.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** pixel, retro, 16-bit, sprite, game, SNES

### pixel_art_8bit
- **Weight:** 0.7–1.0
- **Base:** SD 1.5
- **Description:** Classic 8-bit pixel art style inspired by NES and Game Boy graphics. Minimal pixels with strong color restrictions.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** pixel, retro, 8-bit, NES, game, minimal

### stained_glass
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Stained glass window effect with bold black outlines, translucent colored glass segments, and light shining through.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** stained, glass, window, translucent, gothic, outline

### paper_cut
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Paper cutout craft style with layered paper silhouettes, cast shadows between layers, and flat colored paper textures.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** paper, cutout, craft, layered, shadow, silhouette

### origami_style
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Origami paper folding style with visible creases, geometric folded planes, and matte paper surface. Crisp geometric aesthetic.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** origami, paper, fold, geometric, crease

### embroidery
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Embroidered textile effect with visible stitch patterns, thread texture, and fabric grain. Simulates hand-stitched needlework.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** embroidery, stitch, thread, textile, fabric

### cross_stitch
- **Weight:** 0.6–1.0
- **Base:** SD 1.5
- **Description:** Cross-stitch pattern style with characteristic X-shaped stitches on evenweave fabric. Pixel-like but with thread texture.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** cross-stitch, pattern, thread, fabric, craft

### risograph
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Risograph print effect with soy ink texture, slight misregistration between color layers, and limited CMYK-ish color palette.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** risograph, print, soy, ink, misregistration

### screen_print
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Screen printing / silkscreen effect with bold flat color areas, halftone dots, and characteristic ink overlap lines.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** screen, print, silkscreen, halftone, bold, flat

### lithograph
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Lithographic print effect with grainy stone texture, subtle color separation, and the characteristic matte print quality.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** lithograph, print, stone, grain, matte

## Lighting & Effects

### dramatic_lighting
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Strong chiaroscuro lighting with deep shadows and bright highlights. Creates high-contrast theatrical mood.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** dramatic, chiaroscuro, shadow, contrast, theatrical

### soft_lighting
- **Weight:** 0.4–0.8
- **Base:** SD 1.5
- **Description:** Soft diffused lighting that wraps gently around subjects. Reduces harsh shadows for a flattering, airy feel.
- **Use with:** Any SD 1.5 checkpoint, especially portrait-focused ones
- **Tags:** soft, diffused, gentle, flattering, airy

### neon_glow
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Neon light effects with vibrant glowing tubes, color reflections on surfaces, and cyberpunk-like atmosphere.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** neon, glow, cyberpunk, vibrant, reflection

### volumetric_lighting
- **Weight:** 0.4–0.8
- **Base:** Universal
- **Description:** God rays and volumetric light shafts piercing through scenes. Adds depth with visible light beams through fog or dust.
- **Use with:** Any SD 1.5 or SDXL checkpoint
- **Tags:** volumetric, god rays, light, shaft, depth, fog

### fog_atmosphere
- **Weight:** 0.4–0.8
- **Base:** SD 1.5
- **Description:** Misty and foggy atmospheric effect that adds depth through aerial perspective. Softens distant objects naturally.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** fog, mist, atmosphere, aerial, depth, mood

### lens_flare
- **Weight:** 0.3–0.7
- **Base:** SD 1.5
- **Description:** Camera lens flare effect with anamorphic streaks, ghosting artifacts, and bright light source reflections.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** lens, flare, anamorphic, camera, light

### bokeh_depth
- **Weight:** 0.4–0.8
- **Base:** SD 1.5
- **Description:** Strong bokeh blur effect with creamy out-of-focus areas and circular highlight disks. Creates shallow depth-of-field photography look.
- **Use with:** Photorealistic SD 1.5 checkpoints
- **Tags:** bokeh, blur, depth, shallow, DOF, photography

### chromatic_aberration
- **Weight:** 0.3–0.6
- **Base:** SD 1.5
- **Description:** Chromatic aberration effect with color fringing at edges, simulating cheap or extreme camera lenses. Subtle at low weights.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** chromatic, aberration, fringe, lens, distortion

### film_grain
- **Weight:** 0.3–0.7
- **Base:** Universal
- **Description:** Analog film grain texture that adds organic noise reminiscent of 35mm film stock. Varies by weight from subtle to heavy grain.
- **Use with:** Any SD 1.5 or SDXL checkpoint
- **Tags:** film, grain, analog, noise, 35mm, vintage

### double_exposure
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Double exposure photographic effect blending two overlapping images into one frame. Creates surreal and artistic compositions.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** double, exposure, blend, surreal, photography, artistic

## Environment & Scene

### detailed_background
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Enhances background complexity with additional architectural elements, foliage, objects, and environmental texture.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** background, detail, complex, architecture, environment

### nature_enhance
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Boosts nature scene details with lush foliage, detailed bark, varied leaf colors, and rich vegetation textures.
- **Use with:** Landscape-capable SD 1.5 checkpoints
- **Tags:** nature, foliage, vegetation, lush, green, landscape

### urban_decay
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Abandoned and ruined building aesthetic with peeling paint, broken windows, overgrown vegetation, and weathered surfaces.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** urban, decay, abandoned, ruin, weathered, gritty

### underwater_scene
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Underwater environment with caustic light patterns, floating particles, blue-green color shift, and aquatic atmosphere.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** underwater, caustic, aquatic, blue, marine

### space_nebula
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Cosmic space backgrounds with nebula clouds, star fields, galaxy formations, and vibrant interstellar colors.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** space, nebula, cosmic, galaxy, stars, interstellar

### forest_magic
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Enchanted forest atmosphere with bioluminescent plants, fairy lights, mystical fog, and magical glowing particles.
- **Use with:** Fantasy-oriented SD 1.5 checkpoints
- **Tags:** forest, magic, enchanted, bioluminescent, fairy, mystical

### desert_sand
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Desert environment with sand dunes, sandstorm particles, arid atmosphere, and warm golden tones.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** desert, sand, dune, sandstorm, arid, golden

### snow_winter
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Winter snowfall scene with falling snowflakes, snow-covered surfaces, cold blue tones, and frost effects.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** snow, winter, snowflake, frost, cold, blue

### rain_atmosphere
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Rainy mood with visible rain streaks, wet reflective surfaces, puddles, and overcast grey atmosphere.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** rain, wet, puddle, reflective, moody, overcast

### sunset_golden
- **Weight:** 0.4–0.8
- **Base:** Universal
- **Description:** Golden hour and sunset lighting with warm orange-pink hues, long soft shadows, and rich atmospheric color.
- **Use with:** Any SD 1.5 or SDXL checkpoint
- **Tags:** sunset, golden, hour, warm, orange, pink, lighting

## Photography

### photo_realism
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** General photorealistic enhancement pushing output toward photographic quality. Improves skin texture, lighting, and camera-like composition.
- **Use with:** Semi-realistic to realistic SD 1.5 checkpoints
- **Tags:** photo, realistic, photography, enhancement, quality

### macro_detail
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Macro photography effect with extreme close-up detail, shallow depth of field, and magnified surface textures.
- **Use with:** Photorealistic SD 1.5 checkpoints
- **Tags:** macro, close-up, detail, texture, magnified

### tilt_shift
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Tilt-shift miniature effect making scenes look like small scale models. Selective blur with a sharp central band.
- **Use with:** Landscape and architectural SD 1.5 checkpoints
- **Tags:** tilt-shift, miniature, model, selective, blur

### long_exposure
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Long exposure photography effect with motion blur trails, smoothed water, light streaks, and time-smear artifacts.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** long, exposure, motion, blur, trail, light

### drone_aerial
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Aerial and drone viewpoint looking down at landscapes, buildings, and terrain from elevated perspectives.
- **Use with:** Landscape-capable SD 1.5 checkpoints
- **Tags:** drone, aerial, top-down, bird-eye, landscape

### vintage_photo
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Old vintage photograph look with sepia or faded tones, edge vignetting, paper aging, and slight blur.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** vintage, old, photo, sepia, faded, aged

### polaroid_style
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Polaroid instant photo aesthetic with white bordered frame, slightly washed-out colors, and characteristic exposure quirks.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** polaroid, instant, photo, frame, washed

### hdr_photo
- **Weight:** 0.4–0.8
- **Base:** SD 1.5
- **Description:** High dynamic range photography look with expanded tonal range, recovered shadows, preserved highlights, and vivid local contrast.
- **Use with:** Photorealistic SD 1.5 checkpoints
- **Tags:** HDR, dynamic, range, contrast, vivid, tone

### black_white
- **Weight:** 0.6–1.0
- **Base:** Universal
- **Description:** Monochrome black and white photography conversion with proper luminance mapping, not just desaturation.
- **Use with:** Any SD 1.5 or SDXL checkpoint
- **Tags:** monochrome, black, white, B&W, grayscale

### infrared_photo
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Infrared photography simulation with foliage turning white/pink, dark skies, and characteristic false-color IR palette.
- **Use with:** Photorealistic SD 1.5 checkpoints
- **Tags:** infrared, IR, false-color, foliage, surreal

## Quality & Fix

### quality_adder
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** General quality boost that enhances detail, sharpness, and overall refinement. Good all-purpose quality improvement LoRA.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** quality, enhancement, sharpness, detail, refinement

### lowres_fix
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Fixes low resolution artifacts, compression marks, and blocky patterns. Useful for cleaning up low-quality base outputs.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** lowres, fix, artifact, compression, clean

### color_enhance
- **Weight:** 0.4–0.8
- **Base:** Universal
- **Description:** Makes colors more vivid and saturated without shifting hues. Improves color depth and richness across the entire palette.
- **Use with:** Any SD 1.5 or SDXL checkpoint
- **Tags:** color, enhance, vivid, saturated, rich

### contrast_fix
- **Weight:** 0.4–0.8
- **Base:** SD 1.5
- **Description:** Improves overall contrast by deepening blacks and brightening highlights. Prevents washed-out or flat-looking outputs.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** contrast, fix, black, highlight, depth

### sharp_details
- **Weight:** 0.5–0.9
- **Base:** SD 1.5
- **Description:** Sharpens fine details across the image including textures, edges, and small elements. Combats soft or blurry generation.
- **Use with:** Any SD 1.5 checkpoint
- **Tags:** sharp, detail, edge, texture, crisp
