import dayjs from 'dayjs';
import 'dayjs/locale/ro';

import cookie from 'js-cookie';
import { Notyf } from 'notyf';

import qs from 'qs';

export {Base64} from 'js-base64';

dayjs.locale('ro')

export function makeHljsPlugin(hljs) {
	const Component = {
		props: ["language", "code", "autodetect"],
		data: function() {
		  return {
			detectedLanguage: "",
			unknownLanguage: false
		  };
		},
		computed: {
		  className() {
			if (this.unknownLanguage) return "";

			return "hljs " + this.detectedLanguage;
		  },
		  highlighted() {
			// no idea what language to use, return raw code
			if (!this.autoDetect && !hljs.getLanguage(this.language)) {
			  console.warn(`The language "${this.language}" you specified could not be found.`);
			  this.unknownLanguage = true;
			  return this.code.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#x27;');
			}

			let result = {};
			if (this.autoDetect) {
			  result = hljs.highlightAuto(this.code);
			  this.detectedLanguage = result.language;
			} else {
			  result = hljs.highlight(this.language, this.code, this.ignoreIllegals);
			  this.detectedLanguage = this.language;
			}
			return result.value;
		  },
		  autoDetect() {
			return !this.language || (this.autodetect || this.autodetect === "");
		  },
		  ignoreIllegals() {
			return true;
		  }
		},
		template: `<pre><code :class="className" v-html="highlighted"></code></pre>`
	};
	return Component
}

// Waiting for halfmoon 1.2.0 to launch
// import halfmoon from 'halfmoon';
// window.addEventListener('DOMContentLoaded', () => {
//	halfmoon.onDOMContentLoaded()
//})


// util functions

let notyfConf = {
	duration: 6000,
	ripple: false,
	dismissible: true,
	types: [
		{
			type: 'info',
			background: 'blue',
			icon: {
				className: 'fas fa-info-circle',
				tagName: 'i',
				color: 'white'
			}
		},
		{
			type: 'error',
			background: 'red',
			icon: {
				className: 'fas fa-exclamation-triangle',
				tagName: 'i',
				color: 'white'
			}
		},
		{
			type: 'success',
			background: 'green',
			icon: {
				className: 'fas fa-check-circle',
				tagName: 'i',
				color: 'white'
			}
		},
	]
}

var notyf = undefined;

window.addEventListener('load', () => {
	notyf = new Notyf(notyfConf);
})

/* createToast options
	title: the toast title
	description: the toast description
	status: the toast status (default "info", can be ["success", "error", "info"])
	onclick: onClick handler
*/
let createToast = options => {
	if(notyf === undefined) {
		console.warn("createToast called before window load")
		return
	}

	if (options.status == null) {
		options.status = "info"
	}

	let msg = "";
	if(options.title !== undefined && options.title !== null && options.title !== "") {
		msg += `<h3>${options.title}</h3>`;
	}
	if(options.description !== undefined && options.description !== null && options.description !== "") {
		msg += options.description;
	}

	var notification = notyf.open({
		type: options.status,
		message: msg
	})

	if(options.onclick !== null) {
		notification.on('click', options.onclick)
	}
}

let apiToast = (res, overwrite) => {
	if(overwrite === null || overwrite === undefined) {
		overwrite = {}
	}
	overwrite["status"] = res.status 
	overwrite["description"] = res.data
	createToast(overwrite)
}

let getGradient = (score, maxscore) => {
	let col = "#e3dd71", rap = 0.0;
	if(maxscore != 0 && score != 0) { rap = score / maxscore }	
	if(rap == 1.0) { col = "#7fff00" }
	if(rap < 1.0) { col = "#67cf39" }
	if(rap <= 0.8) { col = "#9fdd2e" }
	if(rap <= 0.6) { col = "#d2eb19" }
	if(rap <= 0.4) { col = "#f1d011" }
	if(rap <= 0.2) { col = "#f6881e" }
	if(rap == 0) { col = "#f11722" }
	return col
}

let getCall = async (call, params) => {
	if(call.startsWith('/')) {
		call = call.substr(1)
	}
	let resp = await fetch(`/api/${call}?${qs.stringify(params)}`, {headers: {'Accept': 'application/json', 'Authorization': cookie.get('kn-sessionid') || "guest"}});
	return await resp.json()
}

let postCall = async (call, params) => {
	if(call.startsWith('/')) {
		call = call.substr(1)
	}
	let resp = await fetch(`/api/${call}`, {
		method: 'POST',
		headers: {'Content-Type': 'application/x-www-form-urlencoded','Accept': 'application/json', 'Authorization': cookie.get('kn-sessionid') || "guest"},
		body: qs.stringify(params)
	});
	return await resp.json();
}

let multipartCall = async (call, formdata) => {
	if(call.startsWith('/')) {
		call = call.substr(1)
	}
	let resp = await fetch(`/api/${call}`, {
		method: 'POST',
		headers: {'Accept': 'application/json', 'Authorization': cookie.get('kn-sessionid') || "guest"},
		body: formdata
	});
	return await resp.json();
}

let getUser = async (name) => {
	return await getCall('/user/getByName', {name: name})
}

let resendEmail = async () => {
	let res = await postCall('/user/resendEmail')	
	apiToast(res)
	return res
}

// backwards compat, TODO should be deleted ASAP
window.createToast = createToast 
window.getGradient = getGradient
window.apiToast = apiToast

window.getCall = getCall
window.postCall = postCall
window.multipartCall = multipartCall
window.getUser = getUser

export { 
	dayjs, cookie, 
	createToast, getGradient, apiToast,
	getCall, postCall, multipartCall, getUser,
	resendEmail
};

// export {halfmoon};
