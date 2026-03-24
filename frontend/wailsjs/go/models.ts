export namespace adb {
	
	export class Device {
	    id: string;
	    status: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.model = source["model"];
	    }
	}
	export class ProcessInfo {
	    pid: string;
	    package: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.package = source["package"];
	    }
	}
	export class UINode {
	    id: string;
	    index: string;
	    text: string;
	    class: string;
	    package: string;
	    contentDesc: string;
	    checkable: string;
	    checked: string;
	    clickable: string;
	    enabled: string;
	    focusable: string;
	    focused: string;
	    scrollable: string;
	    longClickable: string;
	    password: string;
	    selected: string;
	    bounds: string;
	    children: UINode[];
	
	    static createFrom(source: any = {}) {
	        return new UINode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.index = source["index"];
	        this.text = source["text"];
	        this.class = source["class"];
	        this.package = source["package"];
	        this.contentDesc = source["contentDesc"];
	        this.checkable = source["checkable"];
	        this.checked = source["checked"];
	        this.clickable = source["clickable"];
	        this.enabled = source["enabled"];
	        this.focusable = source["focusable"];
	        this.focused = source["focused"];
	        this.scrollable = source["scrollable"];
	        this.longClickable = source["longClickable"];
	        this.password = source["password"];
	        this.selected = source["selected"];
	        this.bounds = source["bounds"];
	        this.children = this.convertValues(source["children"], UINode);
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

