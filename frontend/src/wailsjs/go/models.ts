export namespace catalog {
	
	export class ReadingPosition {
	    progress: number;
	    locator: string;
	    // Go type: time
	    updateat: any;
	
	    static createFrom(source: any = {}) {
	        return new ReadingPosition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.progress = source["progress"];
	        this.locator = source["locator"];
	        this.updateat = this.convertValues(source["updateat"], null);
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
	export class Book {
	    id: number[];
	    title: string;
	    author: string;
	    format: string;
	    fileobjectid: string;
	    coverobjectid: string;
	    filesize: number;
	    position: ReadingPosition;
	    shelves: string[];
	    sha256: string;
	    originalname: string;
	    bookmarks: string[];
	    // Go type: time
	    createdat: any;
	    // Go type: time
	    updatedat: any;
	
	    static createFrom(source: any = {}) {
	        return new Book(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.author = source["author"];
	        this.format = source["format"];
	        this.fileobjectid = source["fileobjectid"];
	        this.coverobjectid = source["coverobjectid"];
	        this.filesize = source["filesize"];
	        this.position = this.convertValues(source["position"], ReadingPosition);
	        this.shelves = source["shelves"];
	        this.sha256 = source["sha256"];
	        this.originalname = source["originalname"];
	        this.bookmarks = source["bookmarks"];
	        this.createdat = this.convertValues(source["createdat"], null);
	        this.updatedat = this.convertValues(source["updatedat"], null);
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

