# Stable Diffusion Models Reference

## Base Architecture Legend

| Base | Full Name | Resolution | Notes |
|------|-----------|------------|-------|
| **SD 1.5** | Stable Diffusion 1.5 | 512x512 | Most compatible, largest LoRA ecosystem, lowest VRAM |
| **SDXL** | Stable Diffusion XL | 1024x1024 | Higher quality out of the box, dual-text-encoder architecture |
| **Pony** | Pony Diffusion V6 XL | 1024x1024 | SDXL-based, strong anime/furry style, versatile fine-tune |
| **Illustrious** | Illustrious XL | 1024x1024 | SDXL-based, optimized for illustration and anime |

---

## Photorealistic Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 1 | realisticVisionV60B1_v51HyperVAE | SD 1.5 | Photorealistic portraits with baked-in HyperVAE for accurate skin tones and lighting | Portraits, headshots, character photos |
| 2 | chilloutmix_v10 | SD 1.5 | Photorealistic Asian portraits with natural skin rendering | Asian portrait photography, lifestyle shots |
| 3 | epicrealism_naturalSinRC1VAE | SD 1.5 | Dramatic cinematic realism with rich contrast and natural color grading | Cinematic stills, dramatic portraits, moody scenes |
| 4 | juggernautXL_ragnarokBy | SDXL | Versatile photorealism with strong detail and coherent anatomy | General photorealism, landscapes, characters, objects |
| 5 | realvisxlV50_v50LightningBakedvae | SDXL | Lightning-optimized fast generation with baked VAE for consistent quality | Quick iterations, real-time preview, batch generation |
| 9 | cyberrealistic_final | SD 1.5 | Hyper-realistic output with extremely fine detail and accurate textures | Ultra-realistic scenes, product-like renders, detailed faces |
| 10 | photon_v1 | SDXL | Cinematic photorealism with film-like color science and depth | Film stills, golden hour, atmospheric photography |

## Anime & Illustration Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 11 | neoMoeDreamxl_v10 | SDXL | Anime style with soft shading and vibrant color palettes | Anime characters, moe style, colorful illustrations |
| 12 | riMixPonyIllustrious_riMixV2 | Pony/Illustrious | Hybrid anime illustration model blending Pony and Illustrious strengths | Anime illustration, character design, mixed style art |
| 13 | veteAnthroAnime_v02 | SD 1.5 | Anime and anthropomorphic character generation | Anthro characters, anime-style furries, hybrid designs |
| 14 | revEnginePonyxl_v10 | Pony XL | Versatile anime generation with strong color and composition | Anime scenes, character art, dynamic poses |
| 6 | realismByStableYogi_ponyV3VAE | Pony | Realistic outputs on Pony base with v3 VAE for improved coherence | Realistic-anime hybrid, semi-realistic characters |
| 7 | realismIllustriousBy_v55FP16 | Illustrious | Photorealistic illustration output from Illustrious base | Realistic illustration, concept art with realistic rendering |
| 8 | illustriousRealismBy_v10VAE | Illustrious | Photorealistic variant v10 with tuned VAE | Realistic anime-adjacent art, detailed illustration |

## Stylized & Artistic Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 15 | t00nBL00MXL_v10 | SDXL | Cartoon and toon style with bold outlines and flat color areas | Cartoon characters, comic toon style, stylized portraits |
| 16 | dreamshaper_8 | SD 1.5 | Fantasy and illustration with painterly quality and rich imagination | Fantasy art, book illustrations, concept art |
| 17 | jaggedBrushworksXL_v10 | SDXL | Artistic painterly style with visible brushstrokes and texture | Oil painting look, impressionist scenes, artistic renders |
| 18 | jrdGreenwhisperXL_v10 | SDXL | Nature and fantasy illustration with organic color palettes | Forest scenes, nature spirits, fantasy landscapes |

## Game Art & Pixel Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 19 | sdxlPixelArt_v20 | SDXL | Pixel art game sprites and tilesets with clean pixel-level control | Game sprites, tilemaps, retro game assets |
| 41 | lowPolyXL_v10 | SDXL | Low poly 3D aesthetic with clean geometric shapes | Low poly game assets, stylized 3D renders, mobile game art |
| 39 | isometricWorld_v10 | SD 1.5 | Isometric game art with consistent 2.5D perspective | Isometric game assets, strategy game buildings, map tiles |

## Traditional Art Style Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 20 | watercolorDream_v10 | SD 1.5 | Watercolor painting style with soft bleeding edges and paper texture | Watercolor illustrations, greeting cards, book art |
| 21 | sketchMasterXL_v10 | SDXL | Pencil and ink sketch with natural stroke variation | Concept sketches, storyboard frames, ink drawings |
| 32 | artNouveauXL_v10 | SDXL | Art Nouveau style with flowing organic lines and ornamental detail | Posters, decorative art, Mucha-style illustrations |
| 38 | stainedGlassXL_v10 | SDXL | Stained glass art with bold lead lines and translucent color fills | Stained glass designs, window mockups, decorative patterns |
| 37 | origamiWorld_v10 | SD 1.5 | Paper craft and origami style with folded paper textures | Origami illustrations, paper craft concepts, creative design |
| 42 | ukiyoDreamXL_v10 | SDXL | Japanese woodblock print style with traditional composition | Ukiyo-e art, Japanese-themed illustrations, poster art |
| 45 | gildedAgeXL_v10 | SDXL | Art Deco and Gilded Age aesthetic with geometric ornament | Art deco posters, vintage luxury design, 1920s aesthetics |
| 44 | pastelSoftXL_v10 | SDXL | Soft pastel style with gentle gradients and muted tones | Children's book art, soft aesthetic, calming visuals |

## 3D & Render Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 22 | renderPersonifyXL_v10 | SDXL | 3D render style with clean materials and studio lighting | Product renders, 3D character mockups, CG scenes |

## Comic & Ink Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 23 | comicInkXL_v10 | SDXL | Comic book ink style with halftone patterns and bold outlines | Comic book pages, graphic novel panels, inked art |

## Cinematic & Photography Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 24 | cineDiffusionXL_v10 | SDXL | Cinematic film stills with anamorphic lens simulation and film grain | Movie stills, cinematic scenes, filmic color grading |
| 25 | natureLensXL_v10 | SDXL | Wildlife and nature photography with accurate animal anatomy | Wildlife shots, nature scenes, outdoor photography |
| 28 | fashionSnapXL_v10 | SDXL | Fashion and editorial photography with studio lighting control | Fashion shoots, editorial layouts, model portfolios |
| 29 | foodPhotoXL_v10 | SDXL | Food photography with realistic textures and appetizing color | Food blog images, menu photography, culinary content |
| 30 | miniWorldXL_v10 | SDXL | Miniature and tilt-shift effect with selective focus | Tilt-shift photography, miniature scenes, dioramas |
| 31 | productShotXL_v10 | SDXL | Product photography with clean backgrounds and studio setup | E-commerce photos, product catalogs, marketing assets |
| 26 | archVisionXL_v10 | SDXL | Architecture and interior photography with accurate perspective | Interior design, architectural visualization, real estate |
| 35 | macroLensXL_v10 | SDXL | Macro and close-up photography with shallow depth of field | Close-up details, texture shots, macro nature photography |

## Sci-Fi & Dark Themes

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 27 | mechaForgeXL_v10 | SDXL | Mecha and robot design with mechanical detail and panel lines | Mecha concepts, robot design, sci-fi vehicles |
| 33 | darkGothicXL_v10 | SDXL | Horror and gothic aesthetic with dark atmosphere and moody lighting | Gothic art, horror scenes, dark fantasy |
| 34 | scifiNebulaXL_v10 | SDXL | Sci-fi and cyberpunk with neon glow and futuristic environments | Cyberpunk scenes, sci-fi cityscapes, futuristic tech |
| 43 | neonNoirXL_v10 | SDXL | Neon noir with high-contrast neon lighting and rain-soaked streets | Noir detective scenes, neon-lit streets, moody cyberpunk |

## Design & Flat Style Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 36 | flatDesignXL_v10 | SDXL | Flat UI and icon design with clean vector-like shapes | App icons, UI elements, flat illustration, infographic art |
| 40 | vaporwaveAesthetics_v10 | SD 1.5 | Vaporwave and retro aesthetic with 80s/90s nostalgia elements | Vaporwave art, retro collages, nostalgic digital art |
