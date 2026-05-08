export namespace main {
	
	export class AppStatus {
	    running: boolean;
	    api_healthy: boolean;
	    version: string;
	    pid: number;
	    config_dir: string;
	
	    static createFrom(source: any = {}) {
	        return new AppStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.api_healthy = source["api_healthy"];
	        this.version = source["version"];
	        this.pid = source["pid"];
	        this.config_dir = source["config_dir"];
	    }
	}
	export class Source {
	    name: string;
	    type: string;
	    path: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.path = source["path"];
	        this.enabled = source["enabled"];
	    }
	}
	export class Config {
	    version: string;
	    sources: Source[];
	    api_port: number;
	    mcp_port: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.sources = this.convertValues(source["sources"], Source);
	        this.api_port = source["api_port"];
	        this.mcp_port = source["mcp_port"];
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
	export class MountStatus {
	    mounted: boolean;
	    mount_point: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new MountStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mounted = source["mounted"];
	        this.mount_point = source["mount_point"];
	        this.error = source["error"];
	    }
	}
	export class QueueStats {
	    pending: number;
	    processing: number;
	    completed: number;
	    failed: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new QueueStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pending = source["pending"];
	        this.processing = source["processing"];
	        this.completed = source["completed"];
	        this.failed = source["failed"];
	        this.total = source["total"];
	    }
	}
	export class QueueStatsResponse {
	    queue_stats: QueueStats;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new QueueStatsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.queue_stats = this.convertValues(source["queue_stats"], QueueStats);
	        this.timestamp = source["timestamp"];
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
	export class SearchResult {
	    id: number;
	    file_id: number;
	    file_path: string;
	    content: string;
	    snippet: string;
	    score: number;
	    metadata: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.file_id = source["file_id"];
	        this.file_path = source["file_path"];
	        this.content = source["content"];
	        this.snippet = source["snippet"];
	        this.score = source["score"];
	        this.metadata = source["metadata"];
	    }
	}
	export class SearchResponse {
	    results: SearchResult[];
	    total: number;
	    query: string;
	    mode: string;
	    time_taken: string;
	    limit: number;
	    offset: number;
	    has_more: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], SearchResult);
	        this.total = source["total"];
	        this.query = source["query"];
	        this.mode = source["mode"];
	        this.time_taken = source["time_taken"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	        this.has_more = source["has_more"];
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

