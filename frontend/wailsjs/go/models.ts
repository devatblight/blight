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
	
	export class CommandDefinition {
	    id: string;
	    title: string;
	    keyword: string;
	    description: string;
	    actionType: string;
	    template: string;
	    keywords?: string[];
	    icon?: string;
	    requiresArgument: boolean;
	    runAsAdmin: boolean;
	    pinned: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CommandDefinition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.keyword = source["keyword"];
	        this.description = source["description"];
	        this.actionType = source["actionType"];
	        this.template = source["template"];
	        this.keywords = source["keywords"];
	        this.icon = source["icon"];
	        this.requiresArgument = source["requiresArgument"];
	        this.runAsAdmin = source["runAsAdmin"];
	        this.pinned = source["pinned"];
	    }
	}
	export class BlightConfig {
	    firstRun: boolean;
	    hotkey: string;
	    maxClipboard: number;
	    indexDirs?: string[];
	    maxResults: number;
	    searchDelay: number;
	    hideWhenDeactivated: boolean;
	    lastQueryMode: string;
	    windowPosition: string;
	    useAnimation: boolean;
	    showPlaceholder: boolean;
	    placeholderText: string;
	    theme: string;
	    footerHints: string;
	    startOnStartup: boolean;
	    hideNotifyIcon: boolean;
	    // Go type: time
	    lastIndexedAt?: any;
	    disableFolderIndex?: boolean;
	    searchEngineURL?: string;
	    aliases?: Record<string, string>;
	    commands?: CommandDefinition[];
	    pinnedItems?: string[];
	
	    static createFrom(source: any = {}) {
	        return new BlightConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.firstRun = source["firstRun"];
	        this.hotkey = source["hotkey"];
	        this.maxClipboard = source["maxClipboard"];
	        this.indexDirs = source["indexDirs"];
	        this.maxResults = source["maxResults"];
	        this.searchDelay = source["searchDelay"];
	        this.hideWhenDeactivated = source["hideWhenDeactivated"];
	        this.lastQueryMode = source["lastQueryMode"];
	        this.windowPosition = source["windowPosition"];
	        this.useAnimation = source["useAnimation"];
	        this.showPlaceholder = source["showPlaceholder"];
	        this.placeholderText = source["placeholderText"];
	        this.theme = source["theme"];
	        this.footerHints = source["footerHints"];
	        this.startOnStartup = source["startOnStartup"];
	        this.hideNotifyIcon = source["hideNotifyIcon"];
	        this.lastIndexedAt = this.convertValues(source["lastIndexedAt"], null);
	        this.disableFolderIndex = source["disableFolderIndex"];
	        this.searchEngineURL = source["searchEngineURL"];
	        this.aliases = source["aliases"];
	        this.commands = this.convertValues(source["commands"], CommandDefinition);
	        this.pinnedItems = source["pinnedItems"];
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
	
	export class ContextAction {
	    id: string;
	    label: string;
	    icon: string;
	    shortcut?: string;
	    destructive?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContextAction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	        this.shortcut = source["shortcut"];
	        this.destructive = source["destructive"];
	    }
	}
	export class SearchResult {
	    id: string;
	    title: string;
	    subtitle: string;
	    icon: string;
	    category: string;
	    path: string;
	    kind: string;
	    primaryActionLabel: string;
	    secondaryActionLabel?: string;
	    supportsActions: boolean;
	
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
	        this.kind = source["kind"];
	        this.primaryActionLabel = source["primaryActionLabel"];
	        this.secondaryActionLabel = source["secondaryActionLabel"];
	        this.supportsActions = source["supportsActions"];
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

