export namespace ipc {
	
	export class Stats {
	    bytes_sent: number;
	    bytes_recv: number;
	    uptime_seconds: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bytes_sent = source["bytes_sent"];
	        this.bytes_recv = source["bytes_recv"];
	        this.uptime_seconds = source["uptime_seconds"];
	    }
	}
	export class Status {
	    state: string;
	    assigned_vip?: string;
	    server_vip?: string;
	    server_addr?: string;
	    helper_version?: string;
	    server_version?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.assigned_vip = source["assigned_vip"];
	        this.server_vip = source["server_vip"];
	        this.server_addr = source["server_addr"];
	        this.helper_version = source["helper_version"];
	        this.server_version = source["server_version"];
	    }
	}

}

export namespace main {
	
	export class IPInfo {
	    query: string;
	    city: string;
	    country: string;
	    isp: string;
	
	    static createFrom(source: any = {}) {
	        return new IPInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.city = source["city"];
	        this.country = source["country"];
	        this.isp = source["isp"];
	    }
	}
	export class InitialConfig {
	    server: string;
	    token: string;
	
	    static createFrom(source: any = {}) {
	        return new InitialConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.token = source["token"];
	    }
	}

}

