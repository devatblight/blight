export namespace files {
	
	export class IndexStatus {
	    state: string;
	    message: string;
	    count: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new IndexStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.message = source["message"];
	        this.count = source["count"];
	        this.total = source["total"];
	    }
	}

}

export namespace main {
	
	export class ContextAction {
	    id: string;
	    label: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new ContextAction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	    }
	}
	export class SearchResult {
	    id: string;
	    title: string;
	    subtitle: string;
	    icon: string;
	    category: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.subtitle = source["subtitle"];
	        this.icon = source["icon"];
	        this.category = source["category"];
	        this.path = source["path"];
	    }
	}
	export class UpdateInfo {
	    available: boolean;
	    version: string;
	    url: string;
	    notes: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.version = source["version"];
	        this.url = source["url"];
	        this.notes = source["notes"];
	        this.error = source["error"];
	    }
	}

}

