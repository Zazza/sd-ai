# LoRA Models Reference for Stable Diffusion

---

## Detail & Quality

### add-detail-xl
- **Weight:** 0.5-0.9
- **Base:** SDXL
- **Description:** Increases overall detail and sharpness in SDXL generation. Enhances textures, edges, and fine elements across the composition.
- **Use with:** SDXL checkpoints (juggernautXL, realvisxl, sdXL_v10VAEFix)
- **Tags:** detail, sharpness, quality, SDXL, texture

### Add_Details_v1.2
- **Weight:** 0.5-0.8
- **Base:** Universal
- **Description:** General detail enhancement adding finer textures and micro-details. Effective on skin, fabric, and environmental surfaces.
- **Use with:** Any checkpoint
- **Tags:** detail, quality, texture, enhancement

### Detail_Tweaker_Illustrious_BSY_V3
- **Weight:** 0.5-1.0
- **Base:** Illustrious XL
- **Description:** Adjusts detail levels specifically for Illustrious XL checkpoints. Positive values enhance, negative values simplify detail.
- **Use with:** Illustrious-based models (illustriousRealismBy, realismIllustriousBy, veloriamixIllustrious)
- **Tags:** detail, tweak, illustrious, adjustment

### ErnieSh4rpD3tails
- **Weight:** 0.5-0.9
- **Base:** SDXL
- **Description:** Sharp detail enhancement with edge emphasis and crisp rendering. Improves clarity and definition.
- **Use with:** SDXL checkpoints
- **Tags:** sharp, detail, edge, crisp, definition

### 500SharpCivitai_Lascalae
- **Weight:** 0.5-0.8
- **Base:** SDXL
- **Description:** High-sharpness LoRA that brings out fine detail and texture clarity. Good for architectural and product shots.
- **Use with:** SDXL photorealistic checkpoints
- **Tags:** sharp, detail, clarity, texture

### polyhedron_new_skin_v1.1
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Improves skin rendering quality with better pore detail, subsurface scattering, and natural skin texture.
- **Use with:** Photorealistic SDXL checkpoints
- **Tags:** skin, detail, realism, pore, subsurface

### st41nedg5CIVIT
- **Weight:** 0.5-0.9
- **Base:** SDXL
- **Description:** Stained and textured edge enhancement adding artistic grain and detail to edges and surfaces.
- **Use with:** SDXL checkpoints, especially artistic ones
- **Tags:** edge, texture, grain, artistic, stain

### to8contrast
- **Weight:** 0.3-0.7
- **Base:** Universal
- **Description:** Boosts contrast for deeper blacks and brighter highlights. Prevents flat or washed-out output.
- **Use with:** Any checkpoint
- **Tags:** contrast, black, highlight, depth

### epi_noiseoffset2
- **Weight:** 0.3-0.7
- **Base:** SD 1.5
- **Description:** Adds noise offset to improve contrast in dark scenes. Produces richer blacks and more realistic low-light rendering.
- **Use with:** Any SD 1.5 checkpoint, especially realistic ones
- **Tags:** noise, offset, dark, contrast, lighting

### Realism Lora By Stable Yogi_V3_Lite
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Lightweight realism enhancement pushing output toward photographic quality. Improves skin texture, lighting, and camera-like composition.
- **Use with:** Semi-realistic to realistic checkpoints
- **Tags:** realism, photo, quality, enhancement

---

## Photography

### HDR_Photography
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** High dynamic range photography look with expanded tonal range, recovered shadows, preserved highlights, and vivid local contrast.
- **Use with:** Photorealistic SDXL checkpoints
- **Tags:** HDR, dynamic, range, contrast, vivid

### photo-realism-flux-v1
- **Weight:** 0.5-0.9
- **Base:** FLUX
- **Description:** Photorealism enhancement specifically tuned for FLUX models. Pushes output toward photographic quality.
- **Use with:** flux_dev
- **Tags:** photo, realism, flux, photography

### SDXL_FILM_PHOTOGRAPHY_STYLE_V1
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Film photography aesthetic with analog grain, natural color science, and characteristic film exposure qualities.
- **Use with:** SDXL checkpoints
- **Tags:** film, photography, analog, grain, cinematic

### Artist_photo_flux
- **Weight:** 0.4-0.8
- **Base:** FLUX
- **Description:** Artistic photography style for FLUX with professional composition and studio-quality lighting.
- **Use with:** flux_dev
- **Tags:** artistic, photography, flux, studio, composition

### Anamorphic film style v2.5
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Cinematic anamorphic lens look with characteristic bokeh, lens flares, and widescreen film quality.
- **Use with:** SDXL photorealistic checkpoints
- **Tags:** anamorphic, film, cinematic, lens, bokeh

### Anamorphic bokeh v2
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Anamorphic bokeh effect with oval-shaped out-of-focus highlights and cinematic depth of field.
- **Use with:** SDXL photorealistic checkpoints
- **Tags:** bokeh, anamorphic, depth, cinematic

### aidmaBokeh-FLUX-V0.1
- **Weight:** 0.3-0.7
- **Base:** FLUX
- **Description:** Bokeh depth-of-field effect for FLUX models with creamy background blur and sharp subject focus.
- **Use with:** flux_dev
- **Tags:** bokeh, blur, depth, flux, focus

### dynamic_shot_42_rim_light
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Dynamic rim lighting creating dramatic backlit silhouettes with glowing edge highlights.
- **Use with:** SDXL checkpoints
- **Tags:** rim, light, backlit, dramatic, silhouette

---

## Portrait & Face Details

### closeupface-v1
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Enhances close-up face rendering with improved facial features, skin detail, and expression accuracy.
- **Use with:** Any checkpoint, especially realistic ones
- **Tags:** face, closeup, portrait, detail, expression

### hairdetailer
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Adds intricate individual hair strands, flyaways, and realistic hair texture for all hair types and lengths.
- **Use with:** Any checkpoint
- **Tags:** hair, strand, detail, texture, flyaway

### real_hair
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Photorealistic hair rendering with natural volume, individual strand detail, and accurate light interaction.
- **Use with:** Photorealistic SDXL checkpoints
- **Tags:** hair, realistic, strand, volume, lighting

### realistic hands
- **Weight:** 0.4-0.8
- **Base:** Universal
- **Description:** Produces photorealistic hands with proper proportions, knuckle detail, and natural skin texture. Works for close-up hand shots.
- **Use with:** Photorealistic checkpoints
- **Tags:** hand, realistic, anatomy, fingers, correction

### Super_Eye_Detailer_By_Stable_Yogi_SDPD0
- **Weight:** 0.5-0.9
- **Base:** SDXL / Pony
- **Description:** Enhances iris patterns, reflections, eyelash detail, and sclera clarity for striking close-up eye shots.
- **Use with:** SDXL and Pony XL checkpoints
- **Tags:** eye, iris, detail, reflection, close-up

---

## Style & Color

### celshading
- **Weight:** 0.5-1.0
- **Base:** Universal
- **Description:** Clean anime cel shading with sharp shadow edges, flat color areas, and minimal gradient. Classic 2D animation look.
- **Use with:** Anime checkpoints
- **Tags:** anime, cel, shading, flat, animation

### zimagebase_flat_color_v2.1
- **Weight:** 0.5-1.0
- **Base:** SDXL
- **Description:** Flat color illustration style removing gradients for clean vector-like aesthetic. Ideal for flat design and illustration.
- **Use with:** SDXL checkpoints
- **Tags:** flat, color, vector, illustration, clean

### coloricher
- **Weight:** 0.3-0.7
- **Base:** SDXL
- **Description:** Color richness enhancement making colors more vivid and saturated without shifting hues.
- **Use with:** SDXL checkpoints
- **Tags:** color, vivid, saturated, rich, enhance

### Black__White
- **Weight:** 0.6-1.0
- **Base:** Universal
- **Description:** Monochrome black and white conversion with proper luminance mapping, not just desaturation.
- **Use with:** Any checkpoint
- **Tags:** monochrome, black, white, B&W, grayscale

### ClassipeintXL2.1
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Classical painting style with rich color blending and painterly texture inspired by fine art traditions.
- **Use with:** SDXL checkpoints
- **Tags:** classical, painting, fine art, painterly

### comicStrips
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Comic strip style with bold outlines, halftone patterns, and panel-ready composition.
- **Use with:** Any checkpoint
- **Tags:** comic, strip, halftone, bold, outline

### CyberPunkAI
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Cyberpunk aesthetic with neon lighting, futuristic tech, and urban dystopian atmosphere.
- **Use with:** SDXL checkpoints
- **Tags:** cyberpunk, neon, futuristic, urban, tech

---

## Traditional Art

### watercolor
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Watercolor painting effect with visible paper texture, color bleeding, and soft wet edges.
- **Use with:** Any checkpoint
- **Tags:** watercolor, painting, paper, bleed, wet

### Watercolor(1)
- **Weight:** 0.5-0.9
- **Base:** SD 1.5
- **Description:** Watercolor style variant with soft brush strokes and traditional watercolor media simulation.
- **Use with:** SD 1.5 checkpoints
- **Tags:** watercolor, painting, soft, traditional

### watercolor_v1_sdxl
- **Weight:** 0.5-0.9
- **Base:** SDXL
- **Description:** Watercolor painting effect specifically tuned for SDXL with rich pigment bleeding and paper grain.
- **Use with:** SDXL checkpoints
- **Tags:** watercolor, SDXL, painting, paper

### Watercolor_V7_E10
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Advanced watercolor style with improved pigment simulation, wet-on-wet effects, and natural paper interaction.
- **Use with:** Any checkpoint
- **Tags:** watercolor, advanced, pigment, wet

### pencil_sketch_illustrious
- **Weight:** 0.6-1.0
- **Base:** Illustrious XL
- **Description:** Graphite pencil sketch style tuned for Illustrious with hatching, shading, and visible pencil marks.
- **Use with:** Illustrious-based models
- **Tags:** pencil, sketch, illustrious, graphite, hatching

### Pencil_Sketch_Style
- **Weight:** 0.6-1.0
- **Base:** Universal
- **Description:** Graphite pencil drawing style with hatching, shading, and visible pencil marks. Produces monochrome sketch-like output.
- **Use with:** Any checkpoint
- **Tags:** pencil, sketch, graphite, monochrome

### Charcoal3.0
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Charcoal drawing effect with smudged lines, rich dark values, and textured paper grain. High contrast and expressive.
- **Use with:** Any checkpoint
- **Tags:** charcoal, drawing, smudge, dark, texture

### Charcoal_Drawing
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Charcoal media simulation with expressive marks, deep blacks, and characteristic smudging quality.
- **Use with:** Any checkpoint
- **Tags:** charcoal, drawing, expressive, smudge

### ink_splats_flux
- **Weight:** 0.4-0.8
- **Base:** FLUX
- **Description:** Ink splatter and splat effects for FLUX with dynamic ink droplets, runs, and accidental ink textures.
- **Use with:** flux_dev
- **Tags:** ink, splat, splatter, flux, dynamic

### Sketch_offcolor
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Off-color sketch style with tinted paper and colored pencil marks creating warm-toned sketch aesthetic.
- **Use with:** Any checkpoint
- **Tags:** sketch, off-color, tinted, warm, pencil

### papercut
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Paper cutout craft style with layered silhouettes, cast shadows between layers, and flat colored paper textures.
- **Use with:** Any checkpoint
- **Tags:** paper, cutout, craft, layered, silhouette

### Risograph_Style_FLUX_by_Ethanar-000001
- **Weight:** 0.4-0.8
- **Base:** FLUX
- **Description:** Risograph print effect with soy ink texture, slight misregistration between color layers, and limited color palette.
- **Use with:** flux_dev
- **Tags:** risograph, print, flux, ink, misregistration

### StainedGlassAI-000006
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Stained glass window effect with bold black outlines, translucent colored glass segments, and light shining through.
- **Use with:** Any checkpoint
- **Tags:** stained, glass, window, translucent, gothic

---

## Pixel Art

### pixel-art-xl-v1.1
- **Weight:** 0.7-1.0
- **Base:** SDXL
- **Description:** Retro pixel art style for SDXL with clean pixel-level control and limited color palettes.
- **Use with:** SDXL checkpoints
- **Tags:** pixel, retro, SDXL, sprite, game

### pixel-Illustrius
- **Weight:** 0.7-1.0
- **Base:** Illustrious XL
- **Description:** Pixel art style tuned for Illustrious base with crisp pixels and game-ready sprite output.
- **Use with:** Illustrious-based models
- **Tags:** pixel, illustrious, retro, sprite, game

### pixel_f2
- **Weight:** 0.7-1.0
- **Base:** Universal
- **Description:** Pixel art variant with clean pixel rendering suitable for game asset creation.
- **Use with:** Any checkpoint
- **Tags:** pixel, retro, game, sprite, asset

---

## Nature & Environment

### Nature SDXL
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Nature scene enhancement with lush foliage, detailed bark, varied leaf colors, and rich vegetation textures.
- **Use with:** SDXL checkpoints
- **Tags:** nature, foliage, vegetation, landscape, SDXL

### Beautiful outdoor-Countryside
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Countryside and rural landscape enhancement with rolling hills, fields, and pastoral atmosphere.
- **Use with:** SDXL checkpoints
- **Tags:** outdoor, countryside, rural, landscape, pastoral

### SDXL_Fog_Sa_May_V2
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Fog and mist atmospheric effect that adds depth through aerial perspective. Softens distant objects naturally.
- **Use with:** SDXL checkpoints
- **Tags:** fog, mist, atmosphere, SDXL, depth

### SDXL_under_water_Sa_May_V1
- **Weight:** 0.4-0.8
- **Base:** SDXL
- **Description:** Underwater environment with caustic light patterns, floating particles, and blue-green color shift.
- **Use with:** SDXL checkpoints
- **Tags:** underwater, caustic, aquatic, blue, SDXL

### underwater-photos
- **Weight:** 0.4-0.8
- **Base:** Universal
- **Description:** Underwater photography style with realistic water effects, light refraction, and aquatic atmosphere.
- **Use with:** Any checkpoint
- **Tags:** underwater, photo, aquatic, water, refraction

### Cloudy_Style
- **Weight:** 0.3-0.7
- **Base:** Universal
- **Description:** Overcast cloudy atmosphere with soft diffused lighting and muted color palette.
- **Use with:** Any checkpoint
- **Tags:** cloud, overcast, diffused, soft, moody

---

## Photography Styles

### Vintage_photo-000018
- **Weight:** 0.4-0.8
- **Base:** Universal
- **Description:** Old vintage photograph look with sepia or faded tones, edge vignetting, paper aging, and slight blur.
- **Use with:** Any checkpoint
- **Tags:** vintage, old, photo, sepia, faded

### NuclearPolaroid1
- **Weight:** 0.4-0.8
- **Base:** Universal
- **Description:** Polaroid instant photo aesthetic with white bordered frame, slightly washed-out colors, and characteristic exposure.
- **Use with:** Any checkpoint
- **Tags:** polaroid, instant, photo, frame, washed

### Infrared_photography_SD15_V2
- **Weight:** 0.4-0.8
- **Base:** SD 1.5
- **Description:** Infrared photography simulation with foliage turning white/pink, dark skies, and false-color IR palette.
- **Use with:** SD 1.5 checkpoints
- **Tags:** infrared, IR, false-color, foliage, surreal

### tilt-shift-v1_1
- **Weight:** 0.6-1.0
- **Base:** Universal
- **Description:** Tilt-shift miniature effect making scenes look like small scale models. Selective blur with sharp central band.
- **Use with:** Landscape and architectural checkpoints
- **Tags:** tilt-shift, miniature, model, selective, blur

---

## Lighting

### Dramatic Lighting Slider
- **Weight:** 0.3-0.8
- **Base:** Universal
- **Description:** Strong chiaroscuro lighting with deep shadows and bright highlights. Creates high-contrast theatrical mood.
- **Use with:** Any checkpoint
- **Tags:** dramatic, chiaroscuro, shadow, contrast, theatrical

---

## Fantasy & Sci-Fi

### MW_Elven_p2_illxl_hybrid
- **Weight:** 0.5-0.9
- **Base:** Illustrious XL
- **Description:** Elven and fantasy character style with pointed ears, ethereal features, and fantasy costume elements.
- **Use with:** Illustrious-based models
- **Tags:** elven, fantasy, ears, ethereal, costume

### DarkestDnDZBase_000004000
- **Weight:** 0.5-0.9
- **Base:** Universal
- **Description:** Dark Dungeons & Dragons aesthetic with medieval fantasy, dark dungeons, and RPG character styling.
- **Use with:** Any checkpoint, especially fantasy-oriented ones
- **Tags:** D&D, dark, fantasy, medieval, RPG
