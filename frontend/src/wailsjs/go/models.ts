export namespace compositor {
	
	export class Position {
	    x: number;
	    y: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class CharacterSlot {
	    name: string;
	    prompt: string;
	    position: Position;
	    scale: number;
	
	    static createFrom(source: any = {}) {
	        return new CharacterSlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.prompt = source["prompt"];
	        this.position = this.convertValues(source["position"], Position);
	        this.scale = source["scale"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class  {
	    name: string;
	    image?: string;
	
	    static createFrom(source: any = {}) {
	        return new (source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.image = source["image"];
	    }
	}
	export class MultiPassResult {
	    image: string;
	    background?: string;
	    characters?: [];
	
	    static createFrom(source: any = {}) {
	        return new MultiPassResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.background = source["background"];
	        this.characters = this.convertValues(source["characters"], );
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Scene {
	    background_prompt: string;
	    negative_prompt: string;
	    refine_prompt?: string;
	    refine_denoise?: number;
	    characters: CharacterSlot[];
	    width: number;
	    height: number;
	    preset_id: number;
	    resolution_id?: number;
	    hires_profile_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new Scene(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.background_prompt = source["background_prompt"];
	        this.negative_prompt = source["negative_prompt"];
	        this.refine_prompt = source["refine_prompt"];
	        this.refine_denoise = source["refine_denoise"];
	        this.characters = this.convertValues(source["characters"], CharacterSlot);
	        this.width = source["width"];
	        this.height = source["height"];
	        this.preset_id = source["preset_id"];
	        this.resolution_id = source["resolution_id"];
	        this.hires_profile_id = source["hires_profile_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace filebrowser {
	
	export class FileEntry {
	    name: string;
	    path: string;
	    is_dir: boolean;
	    size: number;
	    mod_time: string;
	
	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.is_dir = source["is_dir"];
	        this.size = source["size"];
	        this.mod_time = source["mod_time"];
	    }
	}

}

export namespace generation {
	
	export class AnalyzePrompts {
	    system_prompt: string;
	    single_prompt: string;
	    chain_prompts: string[];
	
	    static createFrom(source: any = {}) {
	        return new AnalyzePrompts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.system_prompt = source["system_prompt"];
	        this.single_prompt = source["single_prompt"];
	        this.chain_prompts = source["chain_prompts"];
	    }
	}
	export class DecomposeSceneParams {
	    description: string;
	    preset_id: number;
	
	    static createFrom(source: any = {}) {
	        return new DecomposeSceneParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.preset_id = source["preset_id"];
	    }
	}
	export class GenerateCompoundImageParams {
	    compound_preset_id: number;
	    extra_prompt: string;
	    extra_negative_prompt: string;
	    resolution_id?: number;
	    hires_profile_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateCompoundImageParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.compound_preset_id = source["compound_preset_id"];
	        this.extra_prompt = source["extra_prompt"];
	        this.extra_negative_prompt = source["extra_negative_prompt"];
	        this.resolution_id = source["resolution_id"];
	        this.hires_profile_id = source["hires_profile_id"];
	    }
	}
	export class GenerateFromImageParams {
	    image_base64: string;
	    mode: string;
	    gen_mode: string;
	    preset_id: number;
	    compound_preset_id: number;
	    denoising_strength: number;
	    tags: string;
	    extra_negative_prompt: string;
	    mask_base64: string;
	    mask_blur: number;
	    inpaint_fill: number;
	    inpaint_full_res: boolean;
	    remove_object: boolean;
	    resolution_id?: number;
	    hires_profile_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateFromImageParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image_base64 = source["image_base64"];
	        this.mode = source["mode"];
	        this.gen_mode = source["gen_mode"];
	        this.preset_id = source["preset_id"];
	        this.compound_preset_id = source["compound_preset_id"];
	        this.denoising_strength = source["denoising_strength"];
	        this.tags = source["tags"];
	        this.extra_negative_prompt = source["extra_negative_prompt"];
	        this.mask_base64 = source["mask_base64"];
	        this.mask_blur = source["mask_blur"];
	        this.inpaint_fill = source["inpaint_fill"];
	        this.inpaint_full_res = source["inpaint_full_res"];
	        this.remove_object = source["remove_object"];
	        this.resolution_id = source["resolution_id"];
	        this.hires_profile_id = source["hires_profile_id"];
	    }
	}
	export class GenerateImageParams {
	    preset_id: number;
	    extra_prompt: string;
	    extra_negative_prompt: string;
	    resolution_id?: number;
	    hires_profile_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateImageParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preset_id = source["preset_id"];
	        this.extra_prompt = source["extra_prompt"];
	        this.extra_negative_prompt = source["extra_negative_prompt"];
	        this.resolution_id = source["resolution_id"];
	        this.hires_profile_id = source["hires_profile_id"];
	    }
	}
	export class GenerateImageResult {
	    image: any;
	    parameters: any;
	    info: any;
	    is_preview: boolean;
	    hires_fix_skipped: boolean;
	    hires_fix_manual: boolean;
	    effective_prompt: string;
	    effective_negative_prompt: string;
	
	    static createFrom(source: any = {}) {
	        return new GenerateImageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.parameters = source["parameters"];
	        this.info = source["info"];
	        this.is_preview = source["is_preview"];
	        this.hires_fix_skipped = source["hires_fix_skipped"];
	        this.hires_fix_manual = source["hires_fix_manual"];
	        this.effective_prompt = source["effective_prompt"];
	        this.effective_negative_prompt = source["effective_negative_prompt"];
	    }
	}
	export class GenerateSDPromptParams {
	    preset_id: number;
	    description: string;
	    negative: string;
	
	    static createFrom(source: any = {}) {
	        return new GenerateSDPromptParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preset_id = source["preset_id"];
	        this.description = source["description"];
	        this.negative = source["negative"];
	    }
	}
	export class GenerateSDPromptResult {
	    prompt: string;
	    negative_prompt: string;
	
	    static createFrom(source: any = {}) {
	        return new GenerateSDPromptResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prompt = source["prompt"];
	        this.negative_prompt = source["negative_prompt"];
	    }
	}
	export class RecommendPresetResult {
	    preset_id: number;
	    preset_name: string;
	    extra_prompt: string;
	    reasoning: string;
	
	    static createFrom(source: any = {}) {
	        return new RecommendPresetResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preset_id = source["preset_id"];
	        this.preset_name = source["preset_name"];
	        this.extra_prompt = source["extra_prompt"];
	        this.reasoning = source["reasoning"];
	    }
	}
	export class TestCompoundGenerateParams {
	    selected_ids: number[];
	    prompt: string;
	    negative_prompt: string;
	    resolution_id?: number;
	    hires_profile_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new TestCompoundGenerateParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.selected_ids = source["selected_ids"];
	        this.prompt = source["prompt"];
	        this.negative_prompt = source["negative_prompt"];
	        this.resolution_id = source["resolution_id"];
	        this.hires_profile_id = source["hires_profile_id"];
	    }
	}
	export class TestGenerateParams {
	    mode: string;
	    selected_ids: number[];
	    selected_models: string[];
	    prompt: string;
	    negative_prompt: string;
	    sampler: string;
	    schedule_type: string;
	    steps: number;
	    cfg_scale: number;
	    width: number;
	    height: number;
	    seed?: number;
	    resolution_id?: number;
	    hires_profile_id?: number;
	    init_image?: string;
	
	    static createFrom(source: any = {}) {
	        return new TestGenerateParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.selected_ids = source["selected_ids"];
	        this.selected_models = source["selected_models"];
	        this.prompt = source["prompt"];
	        this.negative_prompt = source["negative_prompt"];
	        this.sampler = source["sampler"];
	        this.schedule_type = source["schedule_type"];
	        this.steps = source["steps"];
	        this.cfg_scale = source["cfg_scale"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.seed = source["seed"];
	        this.resolution_id = source["resolution_id"];
	        this.hires_profile_id = source["hires_profile_id"];
	        this.init_image = source["init_image"];
	    }
	}
	export class TestGenerateResultItem {
	    name: string;
	    image: string;
	    seed: number;
	    error?: string;
	    sampler: string;
	    schedule_type: string;
	    cfg_scale: number;
	    model_name: string;
	
	    static createFrom(source: any = {}) {
	        return new TestGenerateResultItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.image = source["image"];
	        this.seed = source["seed"];
	        this.error = source["error"];
	        this.sampler = source["sampler"];
	        this.schedule_type = source["schedule_type"];
	        this.cfg_scale = source["cfg_scale"];
	        this.model_name = source["model_name"];
	    }
	}
	export class UpscaleImageParams {
	    image_base64: string;
	    gen_info: string;
	    preset_id: number;
	
	    static createFrom(source: any = {}) {
	        return new UpscaleImageParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image_base64 = source["image_base64"];
	        this.gen_info = source["gen_info"];
	        this.preset_id = source["preset_id"];
	    }
	}
	export class UpscalePreviewParams {
	    preview_image_base64: string;
	    preset_id: number;
	    seed: number;
	    denoising_strength?: number;
	    resolution_id?: number;
	    hires_profile_id?: number;
	
	    static createFrom(source: any = {}) {
	        return new UpscalePreviewParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preview_image_base64 = source["preview_image_base64"];
	        this.preset_id = source["preset_id"];
	        this.seed = source["seed"];
	        this.denoising_strength = source["denoising_strength"];
	        this.resolution_id = source["resolution_id"];
	        this.hires_profile_id = source["hires_profile_id"];
	    }
	}

}

export namespace importexport {
	
	export class PresetData {
	    name: string;
	    preset_type: string;
	    type_name: string;
	    prompt: string;
	    negative_prompt: string;
	    sampler: string;
	    schedule_type: string;
	    steps: number;
	    cfg_scale: number;
	    model_name: string;
	    seed?: number;
	    denoising_strength?: number;
	    clip_skip?: number;
	    batch_size?: number;
	    batch_count?: number;
	    vae: string;
	    tags: string;
	    loras: string;
	    source_file?: string;
	
	    static createFrom(source: any = {}) {
	        return new PresetData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.preset_type = source["preset_type"];
	        this.type_name = source["type_name"];
	        this.prompt = source["prompt"];
	        this.negative_prompt = source["negative_prompt"];
	        this.sampler = source["sampler"];
	        this.schedule_type = source["schedule_type"];
	        this.steps = source["steps"];
	        this.cfg_scale = source["cfg_scale"];
	        this.model_name = source["model_name"];
	        this.seed = source["seed"];
	        this.denoising_strength = source["denoising_strength"];
	        this.clip_skip = source["clip_skip"];
	        this.batch_size = source["batch_size"];
	        this.batch_count = source["batch_count"];
	        this.vae = source["vae"];
	        this.tags = source["tags"];
	        this.loras = source["loras"];
	        this.source_file = source["source_file"];
	    }
	}
	export class CompoundStepExportData {
	    step_order: number;
	    denoising_strength: number;
	    preset: PresetData;
	
	    static createFrom(source: any = {}) {
	        return new CompoundStepExportData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step_order = source["step_order"];
	        this.denoising_strength = source["denoising_strength"];
	        this.preset = this.convertValues(source["preset"], PresetData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompoundExportData {
	    name: string;
	    description: string;
	    steps: CompoundStepExportData[];
	
	    static createFrom(source: any = {}) {
	        return new CompoundExportData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.steps = this.convertValues(source["steps"], CompoundStepExportData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompoundImportPreview {
	    pipelines: CompoundExportData[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new CompoundImportPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pipelines = this.convertValues(source["pipelines"], CompoundExportData);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ExportImageParams {
	    image_base64: string;
	    format: string;
	    width: number;
	    height: number;
	    lock_ratio: boolean;
	    quality: number;
	    interpolation: string;
	    filename: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportImageParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image_base64 = source["image_base64"];
	        this.format = source["format"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.lock_ratio = source["lock_ratio"];
	        this.quality = source["quality"];
	        this.interpolation = source["interpolation"];
	        this.filename = source["filename"];
	    }
	}
	export class ImportPreview {
	    presets: PresetData[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.presets = this.convertValues(source["presets"], PresetData);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ValidationWarning {
	    preset_name: string;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new ValidationWarning(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preset_name = source["preset_name"];
	        this.warnings = source["warnings"];
	    }
	}

}

export namespace kids {
	
	export class CategoryInfo {
	    name: string;
	    label: string;
	    alwaysOn: boolean;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CategoryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.alwaysOn = source["alwaysOn"];
	        this.enabled = source["enabled"];
	    }
	}

}

export namespace llm {
	
	export class LLMModel {
	    id: string;
	    object: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.object = source["object"];
	    }
	}

}

export namespace preset {
	
	export class Preset {
	    id: number;
	    name: string;
	    preset_type: string;
	    prompt: string;
	    negative_prompt: string;
	    sampler: string;
	    schedule_type: string;
	    steps: number;
	    cfg_scale: number;
	    model_name: string;
	    seed?: number;
	    denoising_strength?: number;
	    clip_skip?: number;
	    batch_size?: number;
	    batch_count?: number;
	    vae: string;
	    type_id?: number;
	    tags: string;
	    loras: string;
	    is_bundled: boolean;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.preset_type = source["preset_type"];
	        this.prompt = source["prompt"];
	        this.negative_prompt = source["negative_prompt"];
	        this.sampler = source["sampler"];
	        this.schedule_type = source["schedule_type"];
	        this.steps = source["steps"];
	        this.cfg_scale = source["cfg_scale"];
	        this.model_name = source["model_name"];
	        this.seed = source["seed"];
	        this.denoising_strength = source["denoising_strength"];
	        this.clip_skip = source["clip_skip"];
	        this.batch_size = source["batch_size"];
	        this.batch_count = source["batch_count"];
	        this.vae = source["vae"];
	        this.type_id = source["type_id"];
	        this.tags = source["tags"];
	        this.loras = source["loras"];
	        this.is_bundled = source["is_bundled"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class CompoundPresetStep {
	    id: number;
	    compound_preset_id: number;
	    step_order: number;
	    preset_id: number;
	    denoising_strength: number;
	    resolution_id?: number;
	    preset?: Preset;
	
	    static createFrom(source: any = {}) {
	        return new CompoundPresetStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.compound_preset_id = source["compound_preset_id"];
	        this.step_order = source["step_order"];
	        this.preset_id = source["preset_id"];
	        this.denoising_strength = source["denoising_strength"];
	        this.resolution_id = source["resolution_id"];
	        this.preset = this.convertValues(source["preset"], Preset);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CompoundPreset {
	    id: number;
	    name: string;
	    description: string;
	    steps: CompoundPresetStep[];
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new CompoundPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.steps = this.convertValues(source["steps"], CompoundPresetStep);
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ExportPreset {
	    id: number;
	    name: string;
	    format: string;
	    width: number;
	    height: number;
	    lock_ratio: boolean;
	    quality: number;
	    interpolation: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.format = source["format"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.lock_ratio = source["lock_ratio"];
	        this.quality = source["quality"];
	        this.interpolation = source["interpolation"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class HiresProfile {
	    id: number;
	    name: string;
	    upscale: number;
	    denoising_strength: number;
	    upscaler: string;
	    is_builtin: boolean;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new HiresProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.upscale = source["upscale"];
	        this.denoising_strength = source["denoising_strength"];
	        this.upscaler = source["upscaler"];
	        this.is_builtin = source["is_builtin"];
	        this.created_at = source["created_at"];
	    }
	}
	
	export class PresetInstallStatus {
	    id: number;
	    name: string;
	    installed: boolean;
	    missing_sd: string[];
	    missing_lora: string[];
	
	    static createFrom(source: any = {}) {
	        return new PresetInstallStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.installed = source["installed"];
	        this.missing_sd = source["missing_sd"];
	        this.missing_lora = source["missing_lora"];
	    }
	}
	export class PresetType {
	    id: number;
	    name: string;
	    description: string;
	    sort_order: number;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new PresetType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.sort_order = source["sort_order"];
	        this.created_at = source["created_at"];
	    }
	}
	export class Resolution {
	    id: number;
	    name: string;
	    width: number;
	    height: number;
	    is_builtin: boolean;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Resolution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.is_builtin = source["is_builtin"];
	        this.created_at = source["created_at"];
	    }
	}
	export class SavedDescription {
	    id: number;
	    text: string;
	    name: string;
	    negative_prompt: string;
	    type: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new SavedDescription(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.name = source["name"];
	        this.negative_prompt = source["negative_prompt"];
	        this.type = source["type"];
	        this.created_at = source["created_at"];
	    }
	}
	export class SavedPrompt {
	    id: number;
	    text: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new SavedPrompt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.created_at = source["created_at"];
	    }
	}
	export class SavedScene {
	    id: number;
	    name: string;
	    scene_json: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new SavedScene(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.scene_json = source["scene_json"];
	        this.created_at = source["created_at"];
	    }
	}
	export class SessionInfo {
	    id: number;
	    name: string;
	    item_count: number;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.item_count = source["item_count"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class SessionItem {
	    id: number;
	    session_id: number;
	    file_name: string;
	    thumb_name: string;
	    source: string;
	    prompt: string;
	    negative_prompt: string;
	    sampler: string;
	    steps: number;
	    cfg_scale: number;
	    seed?: number;
	    denoising: number;
	    width: number;
	    height: number;
	    is_preview: boolean;
	    preset_id?: number;
	    is_active: boolean;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.session_id = source["session_id"];
	        this.file_name = source["file_name"];
	        this.thumb_name = source["thumb_name"];
	        this.source = source["source"];
	        this.prompt = source["prompt"];
	        this.negative_prompt = source["negative_prompt"];
	        this.sampler = source["sampler"];
	        this.steps = source["steps"];
	        this.cfg_scale = source["cfg_scale"];
	        this.seed = source["seed"];
	        this.denoising = source["denoising"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.is_preview = source["is_preview"];
	        this.preset_id = source["preset_id"];
	        this.is_active = source["is_active"];
	        this.created_at = source["created_at"];
	    }
	}

}

export namespace queue {
	
	export class Job {
	    id: number;
	    type: string;
	    status: string;
	    params: string;
	    progress: number;
	    progress_detail: string;
	    result: string;
	    error: string;
	    source: string;
	    created_at: string;
	    started_at?: string;
	    completed_at?: string;
	    retry_count: number;
	    max_retries: number;
	    next_retry?: string;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.params = source["params"];
	        this.progress = source["progress"];
	        this.progress_detail = source["progress_detail"];
	        this.result = source["result"];
	        this.error = source["error"];
	        this.source = source["source"];
	        this.created_at = source["created_at"];
	        this.started_at = source["started_at"];
	        this.completed_at = source["completed_at"];
	        this.retry_count = source["retry_count"];
	        this.max_retries = source["max_retries"];
	        this.next_retry = source["next_retry"];
	    }
	}

}

export namespace sd {
	
	export class LoRA {
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new LoRA(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}
	export class SDModel {
	    title: string;
	    model_name: string;
	    hash: string;
	    config: string;
	
	    static createFrom(source: any = {}) {
	        return new SDModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.model_name = source["model_name"];
	        this.hash = source["hash"];
	        this.config = source["config"];
	    }
	}
	export class Sampler {
	    name: string;
	    aliases: string[];
	
	    static createFrom(source: any = {}) {
	        return new Sampler(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.aliases = source["aliases"];
	    }
	}
	export class Scheduler {
	    name: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new Scheduler(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	    }
	}
	export class Upscaler {
	    name: string;
	    model_name: string;
	
	    static createFrom(source: any = {}) {
	        return new Upscaler(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.model_name = source["model_name"];
	    }
	}
	export class VAE {
	    model_name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new VAE(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model_name = source["model_name"];
	        this.path = source["path"];
	    }
	}

}

export namespace serverclient {
	
	export class BackendInfo {
	    key: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new BackendInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	    }
	}
	export class CatalogLoRA {
	    name: string;
	    base: string;
	    category: string;
	    default_weight: number;
	    description: string;
	    url?: string;
	    size_mb: number;
	
	    static createFrom(source: any = {}) {
	        return new CatalogLoRA(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.base = source["base"];
	        this.category = source["category"];
	        this.default_weight = source["default_weight"];
	        this.description = source["description"];
	        this.url = source["url"];
	        this.size_mb = source["size_mb"];
	    }
	}
	export class CatalogSDModel {
	    name: string;
	    base: string;
	    category: string;
	    description: string;
	    url?: string;
	    size_gb: number;
	    recommended?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CatalogSDModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.base = source["base"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.url = source["url"];
	        this.size_gb = source["size_gb"];
	        this.recommended = source["recommended"];
	    }
	}
	export class CatalogModel {
	    name: string;
	    description: string;
	    size_gb: number;
	    recommended?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CatalogModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.size_gb = source["size_gb"];
	        this.recommended = source["recommended"];
	    }
	}
	export class Catalog {
	    llm_generate: CatalogModel[];
	    llm_vision: CatalogModel[];
	    sd_models: CatalogSDModel[];
	    lora: CatalogLoRA[];
	
	    static createFrom(source: any = {}) {
	        return new Catalog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.llm_generate = this.convertValues(source["llm_generate"], CatalogModel);
	        this.llm_vision = this.convertValues(source["llm_vision"], CatalogModel);
	        this.sd_models = this.convertValues(source["sd_models"], CatalogSDModel);
	        this.lora = this.convertValues(source["lora"], CatalogLoRA);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class DiscoveredServer {
	    name: string;
	    host: string;
	    port: number;
	    ip_address: string;
	
	    static createFrom(source: any = {}) {
	        return new DiscoveredServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.ip_address = source["ip_address"];
	    }
	}
	export class GPUInfo {
	    name?: string;
	    memory_total_mb?: number;
	    memory_used_mb?: number;
	    memory_free_mb?: number;
	    utilization_percent?: number;
	    available: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GPUInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.memory_total_mb = source["memory_total_mb"];
	        this.memory_used_mb = source["memory_used_mb"];
	        this.memory_free_mb = source["memory_free_mb"];
	        this.utilization_percent = source["utilization_percent"];
	        this.available = source["available"];
	    }
	}
	export class HealthResult {
	    healthy: boolean;
	    latency_ms: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.healthy = source["healthy"];
	        this.latency_ms = source["latency_ms"];
	        this.error = source["error"];
	    }
	}
	export class InstallStatus {
	    key: string;
	    installed: boolean;
	    installing: boolean;
	    progress: string;
	    error?: string;
	    version?: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.installed = source["installed"];
	        this.installing = source["installing"];
	        this.progress = source["progress"];
	        this.error = source["error"];
	        this.version = source["version"];
	    }
	}
	export class LLMModelInfo {
	    name: string;
	    size?: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	    }
	}
	export class ModelInfo {
	    name: string;
	    size: number;
	    extension?: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	        this.extension = source["extension"];
	    }
	}
	export class ProcessStatus {
	    name: string;
	    status: string;
	    pid?: number;
	    uptime?: string;
	    restarts: number;
	    category?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.status = source["status"];
	        this.pid = source["pid"];
	        this.uptime = source["uptime"];
	        this.restarts = source["restarts"];
	        this.category = source["category"];
	    }
	}
	export class ServerModels {
	    sd_checkpoint?: string;
	    llm_running?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ServerModels(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sd_checkpoint = source["sd_checkpoint"];
	        this.llm_running = source["llm_running"];
	    }
	}
	export class ServerStatus {
	    processes: Record<string, ProcessStatus>;
	    health: Record<string, HealthResult>;
	    gpu: GPUInfo;
	    installs: Record<string, InstallStatus>;
	    models: ServerModels;
	
	    static createFrom(source: any = {}) {
	        return new ServerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.processes = this.convertValues(source["processes"], ProcessStatus, true);
	        this.health = this.convertValues(source["health"], HealthResult, true);
	        this.gpu = this.convertValues(source["gpu"], GPUInfo);
	        this.installs = this.convertValues(source["installs"], InstallStatus, true);
	        this.models = this.convertValues(source["models"], ServerModels);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace settings {
	
	export class ServiceInfo {
	    available: boolean;
	    model: string;
	    vision_model?: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.model = source["model"];
	        this.vision_model = source["vision_model"];
	    }
	}
	export class ServiceStatus {
	    llm: ServiceInfo;
	    sd: ServiceInfo;
	
	    static createFrom(source: any = {}) {
	        return new ServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.llm = this.convertValues(source["llm"], ServiceInfo);
	        this.sd = this.convertValues(source["sd"], ServiceInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

