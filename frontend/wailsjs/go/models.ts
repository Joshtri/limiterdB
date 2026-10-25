export namespace domain {
	
	export class DayStat {
	    date: string;
	    listenMin: number;
	    leqDb: number;
	    dosePct: number;
	
	    static createFrom(source: any = {}) {
	        return new DayStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.listenMin = source["listenMin"];
	        this.leqDb = source["leqDb"];
	        this.dosePct = source["dosePct"];
	    }
	}
	export class Device {
	    id: string;
	    name: string;
	    kind: string;
	    muted: boolean;
	    minDb: number;
	    maxDb: number;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.muted = source["muted"];
	        this.minDb = source["minDb"];
	        this.maxDb = source["maxDb"];
	    }
	}
	export class ExposureSummary {
	    today: DayStat;
	    week: DayStat;
	    days: DayStat[];
	
	    static createFrom(source: any = {}) {
	        return new ExposureSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.today = this.convertValues(source["today"], DayStat);
	        this.week = this.convertValues(source["week"], DayStat);
	        this.days = this.convertValues(source["days"], DayStat);
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

export namespace main {
	
	export class BoostView {
	    installed: boolean;
	    linked: boolean;
	    configDir: string;
	
	    static createFrom(source: any = {}) {
	        return new BoostView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.linked = source["linked"];
	        this.configDir = source["configDir"];
	    }
	}
	export class SettingsView {
	    limiterEnabled: boolean;
	    targetSpl: number;
	    ceiling: number;
	    speakerBoost: number;
	    autoStart: boolean;
	    dataDir: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.limiterEnabled = source["limiterEnabled"];
	        this.targetSpl = source["targetSpl"];
	        this.ceiling = source["ceiling"];
	        this.speakerBoost = source["speakerBoost"];
	        this.autoStart = source["autoStart"];
	        this.dataDir = source["dataDir"];
	    }
	}

}

export namespace service {
	
	export class Status {
	    enabled: boolean;
	    active: boolean;
	    currentVolume: number;
	    lastLevel: number;
	    threshold: number;
	    ceiling: number;
	    targetSpl: number;
	    maxSpl: number;
	    calibrated: boolean;
	    heardSpl: number;
	    listening: boolean;
	    deviceOk: boolean;
	    device: domain.Device;
	    boostDb: number;
	    boostError: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.active = source["active"];
	        this.currentVolume = source["currentVolume"];
	        this.lastLevel = source["lastLevel"];
	        this.threshold = source["threshold"];
	        this.ceiling = source["ceiling"];
	        this.targetSpl = source["targetSpl"];
	        this.maxSpl = source["maxSpl"];
	        this.calibrated = source["calibrated"];
	        this.heardSpl = source["heardSpl"];
	        this.listening = source["listening"];
	        this.deviceOk = source["deviceOk"];
	        this.device = this.convertValues(source["device"], domain.Device);
	        this.boostDb = source["boostDb"];
	        this.boostError = source["boostError"];
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

