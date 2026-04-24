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

export namespace main {
	
	export class GenerateImageParams {
	    preset_id: number;
	    extra_prompt: string;
	    extra_negative_prompt: string;
	
	    static createFrom(source: any = {}) {
	        return new GenerateImageParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.preset_id = source["preset_id"];
	        this.extra_prompt = source["extra_prompt"];
	        this.extra_negative_prompt = source["extra_negative_prompt"];
	    }
	}
	export class GenerateImageResult {
	    image: any;
	    parameters: any;
	    info: any;
	
	    static createFrom(source: any = {}) {
	        return new GenerateImageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.parameters = source["parameters"];
	        this.info = source["info"];
	    }
	}
	export class PresetData {
	    name: string;
	    preset_type: string;
	    prompt: string;
	    negative_prompt: string;
	    sampler: string;
	    steps: number;
	    cfg_scale: number;
	    width: number;
	    height: number;
	    model_name: string;
	    seed?: number;
	    denoising_strength?: number;
	    clip_skip?: number;
	    batch_size?: number;
	    batch_count?: number;
	    hires_fix?: boolean;
	    hires_upscale?: number;
	    hires_denoising_strength?: number;
	    hires_upscaler: string;
	    vae: string;
	
	    static createFrom(source: any = {}) {
	        return new PresetData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.preset_type = source["preset_type"];
	        this.prompt = source["prompt"];
	        this.negative_prompt = source["negative_prompt"];
	        this.sampler = source["sampler"];
	        this.steps = source["steps"];
	        this.cfg_scale = source["cfg_scale"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.model_name = source["model_name"];
	        this.seed = source["seed"];
	        this.denoising_strength = source["denoising_strength"];
	        this.clip_skip = source["clip_skip"];
	        this.batch_size = source["batch_size"];
	        this.batch_count = source["batch_count"];
	        this.hires_fix = source["hires_fix"];
	        this.hires_upscale = source["hires_upscale"];
	        this.hires_denoising_strength = source["hires_denoising_strength"];
	        this.hires_upscaler = source["hires_upscaler"];
	        this.vae = source["vae"];
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
	
	export class ServiceStatus {
	    // Go type: struct { Available bool "json:\"available\""; Model string "json:\"model\"" }
	    llm: any;
	    // Go type: struct { Available bool "json:\"available\""; Model string "json:\"model\"" }
	    sd: any;
	
	    static createFrom(source: any = {}) {
	        return new ServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.llm = this.convertValues(source["llm"], Object);
	        this.sd = this.convertValues(source["sd"], Object);
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

export namespace preset {
	
	export class Preset {
	    id: number;
	    name: string;
	    preset_type: string;
	    prompt: string;
	    negative_prompt: string;
	    sampler: string;
	    steps: number;
	    cfg_scale: number;
	    width: number;
	    height: number;
	    model_name: string;
	    seed?: number;
	    denoising_strength?: number;
	    clip_skip?: number;
	    batch_size?: number;
	    batch_count?: number;
	    hires_fix?: boolean;
	    hires_upscale?: number;
	    hires_denoising_strength?: number;
	    hires_upscaler: string;
	    vae: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
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
	        this.steps = source["steps"];
	        this.cfg_scale = source["cfg_scale"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.model_name = source["model_name"];
	        this.seed = source["seed"];
	        this.denoising_strength = source["denoising_strength"];
	        this.clip_skip = source["clip_skip"];
	        this.batch_size = source["batch_size"];
	        this.batch_count = source["batch_count"];
	        this.hires_fix = source["hires_fix"];
	        this.hires_upscale = source["hires_upscale"];
	        this.hires_denoising_strength = source["hires_denoising_strength"];
	        this.hires_upscaler = source["hires_upscaler"];
	        this.vae = source["vae"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class SavedDescription {
	    id: number;
	    text: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new SavedDescription(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class SavedPrompt {
	    id: number;
	    text: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new SavedPrompt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.created_at = this.convertValues(source["created_at"], null);
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

export namespace sd {
	
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

