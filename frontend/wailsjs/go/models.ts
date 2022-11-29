export namespace config {
	
	export class Config {
	    URL?: string;
	    Debug?: boolean;
	    BrowserPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.URL = source["URL"];
	        this.Debug = source["Debug"];
	        this.BrowserPath = source["BrowserPath"];
	    }
	}

}

