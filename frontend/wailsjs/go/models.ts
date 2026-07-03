export namespace config {
	
	export class Forward {
	    type: string;
	    listen: string;
	    target: string;
	
	    static createFrom(source: any = {}) {
	        return new Forward(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.listen = source["listen"];
	        this.target = source["target"];
	    }
	}
	export class Settings {
	    autostart: boolean;
	    theme: string;
	    minimizeToTray: boolean;
	    startMinimized: boolean;
	    hideFromDock: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autostart = source["autostart"];
	        this.theme = source["theme"];
	        this.minimizeToTray = source["minimizeToTray"];
	        this.startMinimized = source["startMinimized"];
	        this.hideFromDock = source["hideFromDock"];
	    }
	}
	export class Tunnel {
	    id: string;
	    name: string;
	    host: string;
	    port: number;
	    user: string;
	    authType: string;
	    password?: string;
	    keyPath?: string;
	    keyPassphrase?: string;
	    forwards: Forward[];
	    autoReconnect: boolean;
	    reconnectMinMs: number;
	    reconnectMaxMs: number;
	    serverAliveInterval: number;
	    status?: string;
	    lastError?: string;
	    // Go type: time
	    startedAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new Tunnel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.authType = source["authType"];
	        this.password = source["password"];
	        this.keyPath = source["keyPath"];
	        this.keyPassphrase = source["keyPassphrase"];
	        this.forwards = this.convertValues(source["forwards"], Forward);
	        this.autoReconnect = source["autoReconnect"];
	        this.reconnectMinMs = source["reconnectMinMs"];
	        this.reconnectMaxMs = source["reconnectMaxMs"];
	        this.serverAliveInterval = source["serverAliveInterval"];
	        this.status = source["status"];
	        this.lastError = source["lastError"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
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

