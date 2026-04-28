# Stable Diffusion Models Reference

## Base Architecture Legend

| Base | Full Name | Resolution | Notes |
|------|-----------|------------|-------|
| **SD 1.5** | Stable Diffusion 1.5 | 512x512 | Most compatible, largest LoRA ecosystem, lowest VRAM |
| **SDXL** | Stable Diffusion XL | 1024x1024 | Higher quality out of the box, dual-text-encoder architecture |
| **Pony XL** | Pony Diffusion V6 XL | 1024x1024 | SDXL-based, strong anime/furry style, versatile fine-tune |
| **Illustrious XL** | Illustrious XL | 1024x1024 | SDXL-based, optimized for illustration and anime |
| **FLUX** | FLUX.1 | 1024x1024 | Next-gen architecture, superior prompt adherence and quality |

---

## Photorealistic Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 1 | absolutereality_v181 | SD 1.5 | High-fidelity photorealism with accurate lighting and material rendering | Realistic portraits, everyday scenes, product shots |
| 2 | realisticVisionV60B1_v51HyperVAE | SD 1.5 | Photorealistic portraits with baked-in HyperVAE for accurate skin tones and lighting | Portraits, headshots, character photos |
| 3 | chilloutmix_v10 | SD 1.5 | Photorealistic Asian portraits with natural skin rendering | Asian portrait photography, lifestyle shots |
| 4 | epicrealism_naturalSinRC1VAE | SD 1.5 | Dramatic cinematic realism with rich contrast and natural color grading | Cinematic stills, dramatic portraits, moody scenes |
| 5 | cyberrealistic_final | SD 1.5 | Hyper-realistic output with extremely fine detail and accurate textures | Ultra-realistic scenes, product-like renders, detailed faces |
| 6 | picxReal_10 | SD 1.5 | Photorealistic with emphasis on natural photo quality and lifelike rendering | Photography-style images, natural-looking portraits |
| 7 | photon_v1 | SDXL | Cinematic photorealism with film-like color science and depth | Film stills, golden hour, atmospheric photography |
| 8 | juggernautXL_ragnarokBy | SDXL | Versatile photorealism with strong detail and coherent anatomy | General photorealism, landscapes, characters, objects |
| 9 | realvisxlV50_v50LightningBakedvae | SDXL | Lightning-optimized fast generation with baked VAE for consistent quality | Quick iterations, real-time preview, batch generation |
| 10 | dreamshaperXL_lightningDPMSDE | SDXL | Lightning-fast SDXL variant of DreamShaper with DPM++ SDE sampling | Fast high-quality generation, rapid prototyping |
| 11 | sdXL_v10VAEFix | SDXL | Base SDXL 1.0 with corrected VAE for stable color output | Baseline SDXL generation, LoRA testing, general purpose |
| 12 | flux_dev | FLUX | Next-generation model with superior prompt understanding and image quality | High-end generation, complex prompts, artistic and realistic output |
| 13 | plantMilkModelSuite_walnut | SDXL | Soft aesthetic photorealism with gentle color palette and smooth rendering | Soft photography, aesthetic portraits, gentle mood |
| 14 | icbinpICantBelieveIts_mid2024 | SDXL | Photorealistic SDXL model with exceptional detail and realism | Photorealistic SDXL output, portraits, scenes |

## Anime & Illustration Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 15 | arthemyAnime_v20 | SD 1.5 | Anime art style with refined linework and vivid coloring | Anime illustrations, character art, vibrant scenes |
| 16 | aetherFaeSemi_version4 | SD 1.5 | Semi-realistic anime with fairy and ethereal aesthetic | Fairy art, ethereal characters, fantasy anime |
| 17 | neoMoeDreamxl_v10 | SDXL | Anime style with soft shading and vibrant color palettes | Anime characters, moe style, colorful illustrations |
| 18 | revAnimated_v2Rebirth | SD 1.5 | Animated illustration style with clean lines and vivid colors | Anime-style illustrations, character design, vibrant art |
| 19 | ghostmix_v20Bakedvae | SD 1.5 | Dark fantasy anime with moody atmosphere and baked VAE | Dark fantasy, gothic anime, atmospheric illustrations |
| 20 | majicmixFantasy_v30Vae | SD 1.5 | Fantasy anime with magical color grading and baked VAE | Fantasy illustrations, magical scenes, vibrant anime |
| 21 | nik0major_v20 | SD 1.5 | Anime illustration with distinctive artistic style | Character art, anime illustration, stylized portraits |
| 22 | babesByStableYogiPony_v60FP16 | Pony XL | Pony-based anime with attractive character rendering | Anime character art, Pony-style illustrations |
| 23 | revEnginePonyxl_v10 | Pony XL | Versatile anime generation with strong color and composition | Anime scenes, character art, dynamic poses |
| 24 | riMixPONYIllustrious_riMixV2 | Pony/Illustrious | Hybrid anime illustration model blending Pony and Illustrious strengths | Anime illustration, character design, mixed style art |
| 25 | t00nBL00MXL_v10 | SDXL | Cartoon and toon style with bold outlines and flat color areas | Cartoon characters, comic toon style, stylized portraits |
| 26 | veloriamixIllustrious_v20 | Illustrious | High-quality anime illustration on Illustrious base | Detailed anime art, illustration, character design |
| 27 | throwingPastaScampi_v20 | SD 1.5 | Quirky anime illustration with expressive style | Expressive character art, fun illustrations |
| 28 | veteAnthroAnime_v02 | SD 1.5 | Anime and anthropomorphic character generation | Anthro characters, anime-style furries, hybrid designs |

## Semi-Realistic & Blended Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 29 | realismByStableYogi_ponyV3VAE | Pony XL | Realistic outputs on Pony base with v3 VAE for improved coherence | Realistic-anime hybrid, semi-realistic characters |
| 30 | realismIllustriousBy_v55FP16 | Illustrious | Photorealistic illustration output from Illustrious base | Realistic illustration, concept art with realistic rendering |
| 31 | illustriousRealismBy_v10VAE | Illustrious | Photorealistic variant with tuned VAE on Illustrious base | Realistic anime-adjacent art, detailed illustration |
| 32 | duelIllAnireal_edgepaintREAL | Illustrious | Hybrid realistic-anime style with painted edge quality | Semi-realistic illustration, painted look, hybrid art |
| 33 | neverendingDreamNED_v122BakedVae | SD 1.5 | Dreamlike semi-realism with soft blending and baked VAE | Dreamy scenes, soft focus art, fantasy realism |
| 34 | fnReal25DNoobxl_v10 | SDXL | 2.5D semi-realistic style bridging anime and realism | 2.5D characters, game-style renders, semi-realistic art |
| 35 | juicyBase_jmBase | SDXL | Versatile base model with balanced anime-realism blend | General purpose, mixed style, balanced output |

## Stylized & Artistic Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 36 | dreamshaper_8 | SD 1.5 | Fantasy and illustration with painterly quality and rich imagination | Fantasy art, book illustrations, concept art |
| 37 | jaggedBrushworksXL_v10 | SDXL | Artistic painterly style with visible brushstrokes and texture | Oil painting look, impressionist scenes, artistic renders |
| 38 | jrdGreenwhisperXL_v10 | SDXL | Nature and fantasy illustration with organic color palettes | Forest scenes, nature spirits, fantasy landscapes |
| 39 | superstyleilxl_carmine | SDXL | Highly stylized illustration with bold carmine tones and strong composition | Stylized illustration, bold art, poster design |
| 40 | gritboundXL_v10 | SDXL | Gritty textured style with raw artistic feel | Gritty illustrations, textured art, rough aesthetic |
| 41 | zavychromaxl_v100 | SDXL | Rich chromatic style with vivid color handling | Colorful art, chromatic illustrations, vibrant scenes |
| 42 | prefectiousXLNSFW_v10 | SDXL | Versatile SDXL with strong style flexibility | Styled generation, creative art, varied aesthetics |
| 43 | FlowingLightRendering_v10 | SD 1.5 | Flowing light effects with luminous rendering quality | Light effects, luminous scenes, glowing elements |

## Game Art & Specialized Models

| # | Model Name | Base | Description | Best For |
|---|-----------|------|-------------|----------|
| 44 | IllustriousNewMecha_v3 | Illustrious | Mecha and mechanical design optimized on Illustrious base | Mecha design, robots, mechanical illustration |
| 45 | edgIncursio_2gbFp16Pruned | SD 1.5 | Compact game-art model with fantasy RPG aesthetic | Game assets, RPG characters, fantasy items |
| 46 | jrdRenderspecXL_jrdRenderspecXLTURBO | SDXL | Rendered specular style with 3D-like material quality | 3D-style renders, product visualization, material art |
| 47 | nightSkyYOZORAStyle_yozoraV1Origin | SD 1.5 | Night sky and starry aesthetic with atmospheric rendering | Night scenes, starry skies, atmospheric landscapes |
| 48 | leosamsHelloworldXL_helloworldXL70 | SDXL | General-purpose SDXL with strong all-around capability | General generation, testing, versatile output |
| 49 | v1-5-pruned-emaonly | SD 1.5 | Base Stable Diffusion 1.5 pruned for minimal VRAM usage | LoRA training, baseline generation, minimum resources |
