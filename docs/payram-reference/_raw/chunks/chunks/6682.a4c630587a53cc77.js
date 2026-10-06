"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[6682],{3658:(t,e,i)=>{i(12936)},12936:(t,e,i)=>{var r=i(88088),a=i(53227),o=i(74897),s=i(94895),n=i(69428);let l=(0,r.AH)`
  :host {
    display: inline-flex !important;
  }

  slot {
    width: 100%;
    display: inline-block;
    font-style: normal;
    font-family: var(--wui-font-family);
    font-feature-settings:
      'tnum' on,
      'lnum' on,
      'case' on;
    line-height: 130%;
    font-weight: var(--wui-font-weight-regular);
    overflow: inherit;
    text-overflow: inherit;
    text-align: var(--local-align);
    color: var(--local-color);
  }

  .wui-line-clamp-1 {
    overflow: hidden;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 1;
  }

  .wui-line-clamp-2 {
    overflow: hidden;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }

  .wui-font-medium-400 {
    font-size: var(--wui-font-size-medium);
    font-weight: var(--wui-font-weight-light);
    letter-spacing: var(--wui-letter-spacing-medium);
  }

  .wui-font-medium-600 {
    font-size: var(--wui-font-size-medium);
    letter-spacing: var(--wui-letter-spacing-medium);
  }

  .wui-font-title-600 {
    font-size: var(--wui-font-size-title);
    letter-spacing: var(--wui-letter-spacing-title);
  }

  .wui-font-title-6-600 {
    font-size: var(--wui-font-size-title-6);
    letter-spacing: var(--wui-letter-spacing-title-6);
  }

  .wui-font-mini-700 {
    font-size: var(--wui-font-size-mini);
    letter-spacing: var(--wui-letter-spacing-mini);
    text-transform: uppercase;
  }

  .wui-font-large-500,
  .wui-font-large-600,
  .wui-font-large-700 {
    font-size: var(--wui-font-size-large);
    letter-spacing: var(--wui-letter-spacing-large);
  }

  .wui-font-2xl-500,
  .wui-font-2xl-600,
  .wui-font-2xl-700 {
    font-size: var(--wui-font-size-2xl);
    letter-spacing: var(--wui-letter-spacing-2xl);
  }

  .wui-font-paragraph-400,
  .wui-font-paragraph-500,
  .wui-font-paragraph-600,
  .wui-font-paragraph-700 {
    font-size: var(--wui-font-size-paragraph);
    letter-spacing: var(--wui-letter-spacing-paragraph);
  }

  .wui-font-small-400,
  .wui-font-small-500,
  .wui-font-small-600 {
    font-size: var(--wui-font-size-small);
    letter-spacing: var(--wui-letter-spacing-small);
  }

  .wui-font-tiny-400,
  .wui-font-tiny-500,
  .wui-font-tiny-600 {
    font-size: var(--wui-font-size-tiny);
    letter-spacing: var(--wui-letter-spacing-tiny);
  }

  .wui-font-micro-700,
  .wui-font-micro-600,
  .wui-font-micro-500 {
    font-size: var(--wui-font-size-micro);
    letter-spacing: var(--wui-letter-spacing-micro);
    text-transform: uppercase;
  }

  .wui-font-tiny-400,
  .wui-font-small-400,
  .wui-font-medium-400,
  .wui-font-paragraph-400 {
    font-weight: var(--wui-font-weight-light);
  }

  .wui-font-large-700,
  .wui-font-paragraph-700,
  .wui-font-micro-700,
  .wui-font-mini-700 {
    font-weight: var(--wui-font-weight-bold);
  }

  .wui-font-medium-600,
  .wui-font-medium-title-600,
  .wui-font-title-6-600,
  .wui-font-large-600,
  .wui-font-paragraph-600,
  .wui-font-small-600,
  .wui-font-tiny-600,
  .wui-font-micro-600 {
    font-weight: var(--wui-font-weight-medium);
  }

  :host([disabled]) {
    opacity: 0.4;
  }
`;var c=function(t,e,i,r){var a,o=arguments.length,s=o<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,i,r);else for(var n=t.length-1;n>=0;n--)(a=t[n])&&(s=(o<3?a(s):o>3?a(e,i,s):a(e,i))||s);return o>3&&s&&Object.defineProperty(e,i,s),s};let u=class extends r.WF{constructor(){super(...arguments),this.variant="paragraph-500",this.color="fg-300",this.align="left",this.lineClamp=void 0}render(){let t={[`wui-font-${this.variant}`]:!0,[`wui-color-${this.color}`]:!0,[`wui-line-clamp-${this.lineClamp}`]:!!this.lineClamp};return this.style.cssText=`
      --local-align: ${this.align};
      --local-color: var(--wui-color-${this.color});
    `,(0,r.qy)`<slot class=${(0,o.H)(t)}></slot>`}};u.styles=[s.W5,l],c([(0,a.MZ)()],u.prototype,"variant",void 0),c([(0,a.MZ)()],u.prototype,"color",void 0),c([(0,a.MZ)()],u.prototype,"align",void 0),c([(0,a.MZ)()],u.prototype,"lineClamp",void 0),u=c([(0,n.E)("wui-text")],u)},24621:(t,e,i)=>{var r=i(88088),a=i(53227),o=i(94895),s=i(21736),n=i(69428);let l=(0,r.AH)`
  :host {
    display: flex;
    width: inherit;
    height: inherit;
  }
`;var c=function(t,e,i,r){var a,o=arguments.length,s=o<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,i,r);else for(var n=t.length-1;n>=0;n--)(a=t[n])&&(s=(o<3?a(s):o>3?a(e,i,s):a(e,i))||s);return o>3&&s&&Object.defineProperty(e,i,s),s};let u=class extends r.WF{render(){return this.style.cssText=`
      flex-direction: ${this.flexDirection};
      flex-wrap: ${this.flexWrap};
      flex-basis: ${this.flexBasis};
      flex-grow: ${this.flexGrow};
      flex-shrink: ${this.flexShrink};
      align-items: ${this.alignItems};
      justify-content: ${this.justifyContent};
      column-gap: ${this.columnGap&&`var(--wui-spacing-${this.columnGap})`};
      row-gap: ${this.rowGap&&`var(--wui-spacing-${this.rowGap})`};
      gap: ${this.gap&&`var(--wui-spacing-${this.gap})`};
      padding-top: ${this.padding&&s.Z.getSpacingStyles(this.padding,0)};
      padding-right: ${this.padding&&s.Z.getSpacingStyles(this.padding,1)};
      padding-bottom: ${this.padding&&s.Z.getSpacingStyles(this.padding,2)};
      padding-left: ${this.padding&&s.Z.getSpacingStyles(this.padding,3)};
      margin-top: ${this.margin&&s.Z.getSpacingStyles(this.margin,0)};
      margin-right: ${this.margin&&s.Z.getSpacingStyles(this.margin,1)};
      margin-bottom: ${this.margin&&s.Z.getSpacingStyles(this.margin,2)};
      margin-left: ${this.margin&&s.Z.getSpacingStyles(this.margin,3)};
    `,(0,r.qy)`<slot></slot>`}};u.styles=[o.W5,l],c([(0,a.MZ)()],u.prototype,"flexDirection",void 0),c([(0,a.MZ)()],u.prototype,"flexWrap",void 0),c([(0,a.MZ)()],u.prototype,"flexBasis",void 0),c([(0,a.MZ)()],u.prototype,"flexGrow",void 0),c([(0,a.MZ)()],u.prototype,"flexShrink",void 0),c([(0,a.MZ)()],u.prototype,"alignItems",void 0),c([(0,a.MZ)()],u.prototype,"justifyContent",void 0),c([(0,a.MZ)()],u.prototype,"columnGap",void 0),c([(0,a.MZ)()],u.prototype,"rowGap",void 0),c([(0,a.MZ)()],u.prototype,"gap",void 0),c([(0,a.MZ)()],u.prototype,"padding",void 0),c([(0,a.MZ)()],u.prototype,"margin",void 0),u=c([(0,n.E)("wui-flex")],u)},33033:(t,e,i)=>{i.d(e,{u:()=>d});var r=i(65972),a=i(34386),o=i(34066),s=i(92929),n=i(18715),l=i(1024),c=i(33452),u=i(37633);class h{constructor(t){this.getNonce=t.getNonce}async createMessage(t){let e={accountAddress:t.accountAddress,chainId:t.chainId,version:"1",domain:"u"<typeof document?"Unknown Domain":document.location.host,uri:"u"<typeof document?"Unknown URI":document.location.href,resources:this.resources,nonce:await this.getNonce(t),issuedAt:this.stringifyDate(new Date),statement:void 0,expirationTime:void 0,notBefore:void 0};return Object.assign(e,{toString:()=>this.stringify(e)})}stringify(t){let e=this.getNetworkName(t.chainId);return[`${t.domain} wants you to sign in with your ${e} account:`,t.accountAddress,t.statement?`
${t.statement}
`:"",`URI: ${t.uri}`,`Version: ${t.version}`,`Chain ID: ${t.chainId}`,`Nonce: ${t.nonce}`,t.issuedAt&&`Issued At: ${t.issuedAt}`,t.expirationTime&&`Expiration Time: ${t.expirationTime}`,t.notBefore&&`Not Before: ${t.notBefore}`,t.requestId&&`Request ID: ${t.requestId}`,t.resources?.length&&t.resources.reduce((t,e)=>`${t}
- ${e}`,"Resources:")].filter(t=>"string"==typeof t).join("\n").trim()}getNetworkName(t){let e=l.W.getAllRequestedCaipNetworks();return u.L.getNetworkNameByCaipNetworkId(e,t)}stringifyDate(t){return t.toISOString()}}class d{constructor(t={}){this.otpUuid=null,this.listeners={sessionChanged:[]},this.localAuthStorageKey=t.localAuthStorageKey||r.Ws.SIWX_AUTH_TOKEN,this.localNonceStorageKey=t.localNonceStorageKey||r.Ws.SIWX_NONCE_TOKEN,this.required=t.required??!0,this.messenger=new h({getNonce:this.getNonce.bind(this)})}async createMessage(t){return this.messenger.createMessage(t)}async addSession(t){let e=await this.request({method:"POST",key:"authenticate",body:{data:t.data,message:t.message,signature:t.signature,clientId:this.getClientId(),walletInfo:this.getWalletInfo()},headers:["nonce","otp"]});this.setStorageToken(e.token,this.localAuthStorageKey),this.emit("sessionChanged",t),this.setAppKitAccountUser(function(t){let e=t.split(".");if(3!==e.length)throw Error("Invalid token");let i=e[1];if("string"!=typeof i)throw Error("Invalid token");let r=i.replace(/-/gu,"+").replace(/_/gu,"/");return JSON.parse(atob(r.padEnd(r.length+(4-r.length%4)%4,"=")))}(e.token)),this.otpUuid=null}async getSessions(t,e){try{if(!this.getStorageToken(this.localAuthStorageKey))return[];let i=await this.request({method:"GET",key:"me",query:{},headers:["auth"]});if(!i)return[];let r=i.address.toLowerCase()===e.toLowerCase(),a=i.caip2Network===t;if(!r||!a)return[];let o={data:{accountAddress:i.address,chainId:i.caip2Network},message:"",signature:""};return this.emit("sessionChanged",o),this.setAppKitAccountUser(i),[o]}catch{return[]}}async revokeSession(t,e){return Promise.resolve(this.clearStorageTokens())}async setSessions(t){if(0===t.length)this.clearStorageTokens();else{let e=t.find(t=>t.data.chainId===(0,c.kg)()?.caipNetworkId)||t[0];await this.addSession(e)}}getRequired(){return this.required}async getSessionAccount(){if(!this.getStorageToken(this.localAuthStorageKey))throw Error("Not authenticated");return this.request({method:"GET",key:"me",body:void 0,query:{includeAppKitAccount:!0},headers:["auth"]})}async setSessionAccountMetadata(t=null){if(!this.getStorageToken(this.localAuthStorageKey))throw Error("Not authenticated");return this.request({method:"PUT",key:"account-metadata",body:{metadata:t},headers:["auth"]})}on(t,e){return this.listeners[t].push(e),()=>{this.listeners[t]=this.listeners[t].filter(t=>t!==e)}}removeAllListeners(){Object.keys(this.listeners).forEach(t=>{this.listeners[t]=[]})}async requestEmailOtp({email:t,account:e}){let i=await this.request({method:"POST",key:"otp",body:{email:t,account:e}});return this.otpUuid=i.uuid,this.messenger.resources=[`email:${t}`],i}confirmEmailOtp({code:t}){return this.request({method:"PUT",key:"otp",body:{code:t},headers:["otp"]})}async request({method:t,key:e,query:i,body:r,headers:o}){let{projectId:s,st:n,sv:l}=this.getSDKProperties(),c=new URL(`${a.o.W3M_API_URL}/auth/v1/${String(e)}`);c.searchParams.set("projectId",s),c.searchParams.set("st",n),c.searchParams.set("sv",l),i&&Object.entries(i).forEach(([t,e])=>c.searchParams.set(t,String(e)));let u=await fetch(c,{method:t,body:r?JSON.stringify(r):void 0,headers:Array.isArray(o)?o.reduce((t,e)=>{switch(e){case"nonce":t["x-nonce-jwt"]=`Bearer ${this.getStorageToken(this.localNonceStorageKey)}`;break;case"auth":t.Authorization=`Bearer ${this.getStorageToken(this.localAuthStorageKey)}`;break;case"otp":this.otpUuid&&(t["x-otp"]=this.otpUuid)}return t},{}):void 0});if(!u.ok)throw Error(await u.text());return u.headers.get("content-type")?.includes("application/json")?u.json():null}getStorageToken(t){return r.Ud.getItem(t)}setStorageToken(t,e){r.Ud.setItem(e,t)}clearStorageTokens(){this.otpUuid=null,r.Ud.removeItem(this.localAuthStorageKey),r.Ud.removeItem(this.localNonceStorageKey),this.emit("sessionChanged",void 0)}async getNonce(){let{nonce:t,token:e}=await this.request({method:"GET",key:"nonce"});return this.setStorageToken(e,this.localNonceStorageKey),t}getClientId(){return n.T.state.clientId}getWalletInfo(){let{connectedWalletInfo:t}=o.U.state;if(!t)return;if("social"in t)return{type:"social",social:t.social,identifier:t.identifier};let{name:e,icon:i}=t,r="unknown";switch(t.type){case"EXTERNAL":case"INJECTED":case"ANNOUNCED":r="extension";break;case"WALLET_CONNECT":r="walletconnect";break;default:r="unknown"}return{type:r,name:e,icon:i}}getSDKProperties(){return s.N._getSdkProperties()}emit(t,e){this.listeners[t].forEach(t=>t(e))}setAppKitAccountUser(t){let{email:e}=t;e&&Object.values(a.o.CHAIN).forEach(t=>{l.W.setAccountProp("user",{email:e},t)})}}},40859:(t,e,i)=>{i.d(e,{J:()=>a});var r=i(87850);let a=t=>t??r.s6},42608:(t,e,i)=>{i.d(e,{M:()=>o});var r=i(36424);let a={attribute:!0,type:String,converter:r.W3,reflect:!1,hasChanged:r.Ec};function o(t){return(e,i)=>{let r;return"object"==typeof i?((t=a,e,i)=>{let{kind:r,metadata:o}=i,s=globalThis.litPropertyMetadata.get(o);if(void 0===s&&globalThis.litPropertyMetadata.set(o,s=new Map),"setter"===r&&((t=Object.create(t)).wrapped=!0),s.set(i.name,t),"accessor"===r){let{name:r}=i;return{set(i){let a=e.get.call(this);e.set.call(this,i),this.requestUpdate(r,a,t,!0,i)},init(e){return void 0!==e&&this.C(r,void 0,t,e),e}}}if("setter"===r){let{name:r}=i;return function(i){let a=this[r];e.call(this,i),this.requestUpdate(r,a,t,!0,i)}}throw Error("Unsupported decorator location: "+r)})(t,e,i):(r=e.hasOwnProperty(i),e.constructor.createProperty(i,t),r?Object.getOwnPropertyDescriptor(e,i):void 0)}}},53227:(t,e,i)=>{i.d(e,{MZ:()=>r.M,wk:()=>a.w});var r=i(42608),a=i(97734)},70508:(t,e,i)=>{i.d(e,{OA:()=>r,WL:()=>o,u$:()=>a});let r={ATTRIBUTE:1,CHILD:2,PROPERTY:3,BOOLEAN_ATTRIBUTE:4,EVENT:5,ELEMENT:6},a=t=>(...e)=>({_$litDirective$:t,values:e});class o{constructor(t){}get _$AU(){return this._$AM._$AU}_$AT(t,e,i){this._$Ct=t,this._$AM=e,this._$Ci=i}_$AS(t,e){return this.update(t,e)}update(t,e){return this.render(...e)}}},74897:(t,e,i)=>{i.d(e,{H:()=>o});var r=i(87850),a=i(70508);let o=(0,a.u$)(class extends a.WL{constructor(t){if(super(t),t.type!==a.OA.ATTRIBUTE||"class"!==t.name||t.strings?.length>2)throw Error("`classMap()` can only be used in the `class` attribute and must be the only part in the attribute.")}render(t){return" "+Object.keys(t).filter(e=>t[e]).join(" ")+" "}update(t,[e]){if(void 0===this.st){for(let i in this.st=new Set,void 0!==t.strings&&(this.nt=new Set(t.strings.join(" ").split(/\s/).filter(t=>""!==t))),e)e[i]&&!this.nt?.has(i)&&this.st.add(i);return this.render(e)}let i=t.element.classList;for(let t of this.st)t in e||(i.remove(t),this.st.delete(t));for(let t in e){let r=!!e[t];r===this.st.has(t)||this.nt?.has(t)||(r?(i.add(t),this.st.add(t)):(i.remove(t),this.st.delete(t)))}return r.c0}})},74970:(t,e,i)=>{i(24621)},84915:(t,e,i)=>{var r=i(88088),a=i(53227),o=i(94895),s=i(69428);let n=(0,r.AH)`
  :host {
    display: flex;
  }

  :host([data-size='sm']) > svg {
    width: 12px;
    height: 12px;
  }

  :host([data-size='md']) > svg {
    width: 16px;
    height: 16px;
  }

  :host([data-size='lg']) > svg {
    width: 24px;
    height: 24px;
  }

  :host([data-size='xl']) > svg {
    width: 32px;
    height: 32px;
  }

  svg {
    animation: rotate 2s linear infinite;
  }

  circle {
    fill: none;
    stroke: var(--local-color);
    stroke-width: 4px;
    stroke-dasharray: 1, 124;
    stroke-dashoffset: 0;
    stroke-linecap: round;
    animation: dash 1.5s ease-in-out infinite;
  }

  :host([data-size='md']) > svg > circle {
    stroke-width: 6px;
  }

  :host([data-size='sm']) > svg > circle {
    stroke-width: 8px;
  }

  @keyframes rotate {
    100% {
      transform: rotate(360deg);
    }
  }

  @keyframes dash {
    0% {
      stroke-dasharray: 1, 124;
      stroke-dashoffset: 0;
    }

    50% {
      stroke-dasharray: 90, 124;
      stroke-dashoffset: -35;
    }

    100% {
      stroke-dashoffset: -125;
    }
  }
`;var l=function(t,e,i,r){var a,o=arguments.length,s=o<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,i,r);else for(var n=t.length-1;n>=0;n--)(a=t[n])&&(s=(o<3?a(s):o>3?a(e,i,s):a(e,i))||s);return o>3&&s&&Object.defineProperty(e,i,s),s};let c=class extends r.WF{constructor(){super(...arguments),this.color="accent-100",this.size="lg"}render(){return this.style.cssText=`--local-color: ${"inherit"===this.color?"inherit":`var(--wui-color-${this.color})`}`,this.dataset.size=this.size,(0,r.qy)`<svg viewBox="25 25 50 50">
      <circle r="20" cy="50" cx="50"></circle>
    </svg>`}};c.styles=[o.W5,n],l([(0,a.MZ)()],c.prototype,"color",void 0),l([(0,a.MZ)()],c.prototype,"size",void 0),c=l([(0,s.E)("wui-loading-spinner")],c)},85278:(t,e,i)=>{i.r(e),i.d(e,{W3mDataCaptureOtpConfirmView:()=>x,W3mDataCaptureView:()=>S,W3mEmailSuffixesWidget:()=>c,W3mRecentEmailsWidget:()=>d});var r=i(88088),a=i(53227),o=i(59306);let s=(0,r.AH)`
  .email-sufixes {
    display: flex;
    flex-direction: row;
    gap: var(--wui-spacing-3xs);
    overflow-x: auto;
    max-width: 100%;
    margin-top: var(--wui-spacing-s);
    margin-bottom: calc(-1 * var(--wui-spacing-m));
    padding-bottom: var(--wui-spacing-m);
    margin-left: calc(-1 * var(--wui-spacing-m));
    margin-right: calc(-1 * var(--wui-spacing-m));
    padding-left: var(--wui-spacing-m);
    padding-right: var(--wui-spacing-m);

    &::-webkit-scrollbar {
      display: none;
    }
  }
`;var n=function(t,e,i,r){var a,o=arguments.length,s=o<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,i,r);else for(var n=t.length-1;n>=0;n--)(a=t[n])&&(s=(o<3?a(s):o>3?a(e,i,s):a(e,i))||s);return o>3&&s&&Object.defineProperty(e,i,s),s};let l=["@gmail.com","@outlook.com","@yahoo.com","@hotmail.com","@aol.com","@icloud.com","@zoho.com"],c=class extends r.WF{constructor(){super(...arguments),this.email=""}render(){let t=l.filter(this.filter.bind(this)).map(this.item.bind(this));return 0===t.length?null:(0,r.qy)`<div class="email-sufixes">${t}</div>`}filter(t){if(!this.email)return!1;let e=this.email.split("@");if(e.length<2)return!0;let i=e.pop();return t.includes(i)&&t!==`@${i}`}item(t){let e=()=>{let e=this.email.split("@");e.length>1&&e.pop();let i=e[0]+t;this.dispatchEvent(new CustomEvent("change",{detail:i,bubbles:!0,composed:!0}))};return(0,r.qy)`<wui-button variant="neutral" size="sm" @click=${e}
      >${t}</wui-button
    >`}};c.styles=[s],n([(0,a.MZ)()],c.prototype,"email",void 0),c=n([(0,o.EM)("w3m-email-suffixes-widget")],c);let u=(0,r.AH)`
  .recent-emails {
    display: flex;
    flex-direction: column;
    padding: var(--wui-spacing-s) 0;
    border-top: 1px solid var(--wui-color-gray-glass-005);
    border-bottom: 1px solid var(--wui-color-gray-glass-005);
  }

  .recent-emails-heading {
    margin-bottom: var(--wui-spacing-s);
  }

  .recent-emails-list-item {
    --wui-color-gray-glass-002: transparent;
  }
`;var h=function(t,e,i,r){var a,o=arguments.length,s=o<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,i,r);else for(var n=t.length-1;n>=0;n--)(a=t[n])&&(s=(o<3?a(s):o>3?a(e,i,s):a(e,i))||s);return o>3&&s&&Object.defineProperty(e,i,s),s};let d=class extends r.WF{constructor(){super(...arguments),this.emails=[]}render(){return 0===this.emails.length?null:(0,r.qy)`<div class="recent-emails">
      <wui-text variant="micro-600" color="fg-200" class="recent-emails-heading"
        >Recently used emails</wui-text
      >
      ${this.emails.map(this.item.bind(this))}
    </div>`}item(t){let e=()=>{this.dispatchEvent(new CustomEvent("select",{detail:t,bubbles:!0,composed:!0}))};return(0,r.qy)`<wui-list-item
      @click=${e}
      ?chevron=${!0}
      icon="mail"
      iconVariant="overlay"
      class="recent-emails-list-item"
    >
      <wui-text variant="paragraph-500" color="fg-100">${t}</wui-text>
    </wui-list-item>`}};d.styles=[u],h([(0,a.MZ)()],d.prototype,"emails",void 0),d=h([(0,o.EM)("w3m-recent-emails-widget")],d);var p=i(19392),g=i(44458),f=i(1024),m=i(44963),w=i(33033),y=i(59047),v=function(t,e,i,r){var a,o=arguments.length,s=o<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,i,r);else for(var n=t.length-1;n>=0;n--)(a=t[n])&&(s=(o<3?a(s):o>3?a(e,i,s):a(e,i))||s);return o>3&&s&&Object.defineProperty(e,i,s),s};let x=class extends y.H{constructor(){super(...arguments),this.siwx=p.H.state.siwx,this.onOtpSubmit=async t=>{await this.siwx.confirmEmailOtp({code:t}),g.I.replace("SIWXSignMessage")},this.onOtpResend=async t=>{let e=f.W.getAccountData();if(!e?.caipAddress)throw Error("No account data found");await this.siwx.requestEmailOtp({email:t,account:e.caipAddress})}}connectedCallback(){this.siwx&&this.siwx instanceof w.u||m.P.showError("ReownAuthentication is not initialized."),super.connectedCallback()}shouldSubmitOnOtpChange(){return this.otp.length===y.H.OTP_LENGTH}};v([(0,a.wk)()],x.prototype,"siwx",void 0),x=v([(0,o.EM)("w3m-data-capture-otp-confirm-view")],x);var b=i(65972);let $=(0,r.AH)`
  .hero {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--wui-spacing-3xs);

    transition-property: margin, height;
    transition-duration: var(--wui-duration-md);
    transition-timing-function: var(--wui-ease-out-power-1);
    margin-top: -100px;

    &[data-state='loading'] {
      margin-top: 0px;
    }

    position: relative;
    &:after {
      content: '';
      position: absolute;
      bottom: 0;
      height: 252px;
      width: 360px;
      background: radial-gradient(
        96.11% 53.95% at 50% 51.28%,
        transparent 0%,
        color-mix(in srgb, var(--wui-color-bg-100) 5%, transparent) 49%,
        color-mix(in srgb, var(--wui-color-bg-100) 65%, transparent) 99.43%
      );
    }
  }

  .hero-main-icon {
    width: 176px;
    transition-property: background-color;
    transition-duration: var(--wui-duration-lg);
    transition-timing-function: var(--wui-ease-out-power-1);

    &[data-state='loading'] {
      width: 56px;
    }
  }

  .hero-row {
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: center;
    gap: var(--wui-spacing-3xs);
    flex-wrap: nowrap;
    min-width: fit-content;

    &:nth-child(1) {
      transform: translateX(-30px);
    }

    &:nth-child(2) {
      transform: translateX(30px);
    }

    &:nth-child(4) {
      transform: translateX(40px);
    }

    transition-property: height;
    transition-duration: var(--wui-duration-md);
    transition-timing-function: var(--wui-ease-out-power-1);
    height: 68px;

    &[data-state='loading'] {
      height: 0px;
    }
  }

  .hero-row-icon {
    opacity: 0.1;
    transition-property: opacity;
    transition-duration: var(--wui-duration-md);
    transition-timing-function: var(--wui-ease-out-power-1);

    &[data-state='loading'] {
      opacity: 0;
    }
  }
`;var k=function(t,e,i,r){var a,o=arguments.length,s=o<3?e:null===r?r=Object.getOwnPropertyDescriptor(e,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,i,r);else for(var n=t.length-1;n>=0;n--)(a=t[n])&&(s=(o<3?a(s):o>3?a(e,i,s):a(e,i))||s);return o>3&&s&&Object.defineProperty(e,i,s),s};let S=class extends r.WF{constructor(){super(...arguments),this.email=g.I.state.data?.email??f.W.getAccountData()?.user?.email??"",this.address=f.W.getAccountData()?.address??"",this.loading=!1,this.appName=p.H.state.metadata?.name??"AppKit",this.siwx=p.H.state.siwx,this.isRequired=Array.isArray(p.H.state.remoteFeatures?.emailCapture)&&p.H.state.remoteFeatures?.emailCapture.includes("required"),this.recentEmails=this.getRecentEmails()}connectedCallback(){this.siwx&&this.siwx instanceof w.u||m.P.showError("ReownAuthentication is not initialized. Please contact support."),super.connectedCallback()}firstUpdated(){this.loading=!1,this.recentEmails=this.getRecentEmails(),this.email&&this.onSubmit()}render(){return(0,r.qy)`
      <wui-flex flexDirection="column" .padding=${["3xs","m","m","m"]} gap="l">
        ${this.hero()} ${this.paragraph()} ${this.emailInput()} ${this.recentEmailsWidget()}
        ${this.footerActions()}
      </wui-flex>
    `}hero(){return(0,r.qy)`
      <div class="hero" data-state=${this.loading?"loading":"default"}>
        ${this.heroRow(["id","mail","wallet","x","solana","qrCode"])}
        ${this.heroRow(["mail","farcaster","wallet","discord","mobile","qrCode"])}
        <div class="hero-row">
          ${this.heroIcon("github")} ${this.heroIcon("bank")}
          <wui-icon-box
            size="xl"
            iconSize="xxl"
            iconColor=${this.loading?"fg-100":"accent-100"}
            backgroundColor=${this.loading?"fg-100":"accent-100"}
            icon=${this.loading?"id":"user"}
            isOpaque
            class="hero-main-icon"
            data-state=${this.loading?"loading":"default"}
          >
          </wui-icon-box>
          ${this.heroIcon("id")} ${this.heroIcon("card")}
        </div>
        ${this.heroRow(["google","id","github","verify","apple","mobile"])}
      </div>
    `}heroRow(t){return(0,r.qy)`
      <div class="hero-row" data-state=${this.loading?"loading":"default"}>
        ${t.map(this.heroIcon.bind(this))}
      </div>
    `}heroIcon(t){return(0,r.qy)`
      <wui-icon-box
        size="xl"
        iconSize="xxl"
        iconColor="fg-100"
        backgroundColor="fg-100"
        icon=${t}
        data-state=${this.loading?"loading":"default"}
        isOpaque
        class="hero-row-icon"
      >
      </wui-icon-box>
    `}paragraph(){return this.loading?(0,r.qy)`
        <wui-text variant="paragraph-400" color="fg-200" align="center"
          >We are verifying your account with email
          <wui-text variant="paragraph-600" color="accent-100">${this.email}</wui-text> and address
          <wui-text variant="paragraph-600" color="fg-100">
            ${o.Zv.getTruncateString({string:this.address,charsEnd:4,charsStart:4,truncate:"middle"})} </wui-text
          >, please wait a moment.</wui-text
        >
      `:this.isRequired?(0,r.qy)`
        <wui-text variant="paragraph-600" color="fg-100" align="center">
          ${this.appName} requires your email for authentication.
        </wui-text>
      `:(0,r.qy)`
      <wui-flex flexDirection="column" gap="xs" alignItems="center">
        <wui-text variant="paragraph-600" color="fg-100" align="center" size>
          ${this.appName} would like to collect your email.
        </wui-text>

        <wui-text variant="small-400" color="fg-200" align="center">
          Don't worry, it's optional&mdash;you can skip this step.
        </wui-text>
      </wui-flex>
    `}emailInput(){if(this.loading)return null;let t=t=>{"Enter"===t.key&&this.onSubmit()},e=t=>{this.email=t.detail};return(0,r.qy)`
      <wui-flex flexDirection="column">
        <wui-email-input
          .value=${this.email}
          .disabled=${this.loading}
          @inputChange=${e}
          @keydown=${t}
        ></wui-email-input>

        <w3m-email-suffixes-widget
          .email=${this.email}
          @change=${e}
        ></w3m-email-suffixes-widget>
      </wui-flex>
    `}recentEmailsWidget(){if(0===this.recentEmails.length||this.loading)return null;let t=t=>{this.email=t.detail,this.onSubmit()};return(0,r.qy)`
      <w3m-recent-emails-widget
        .emails=${this.recentEmails}
        @select=${t}
      ></w3m-recent-emails-widget>
    `}footerActions(){return(0,r.qy)`
      <wui-flex flexDirection="row" fullWidth gap="s">
        ${this.isRequired?null:(0,r.qy)`<wui-button
              size="lg"
              variant="neutral"
              fullWidth
              .disabled=${this.loading}
              @click=${this.onSkip.bind(this)}
              >Skip this step</wui-button
            >`}

        <wui-button
          size="lg"
          variant="main"
          type="submit"
          fullWidth
          .disabled=${!this.email||!this.isValidEmail(this.email)}
          .loading=${this.loading}
          @click=${this.onSubmit.bind(this)}
        >
          Continue
        </wui-button>
      </wui-flex>
    `}async onSubmit(){if(!(this.siwx instanceof w.u))return void m.P.showError("ReownAuthentication is not initialized. Please contact support.");let t=f.W.getActiveCaipAddress();if(!t)throw Error("Account is not connected.");if(!this.isValidEmail(this.email))return void m.P.showError("Please provide a valid email.");try{this.loading=!0;let e=await this.siwx.requestEmailOtp({email:this.email,account:t});this.pushRecentEmail(this.email),null===e.uuid?g.I.replace("SIWXSignMessage"):g.I.replace("DataCaptureOtpConfirm",{email:this.email})}catch(t){m.P.showError("Failed to send email OTP"),this.loading=!1}}onSkip(){g.I.replace("SIWXSignMessage")}getRecentEmails(){let t=b.Ud.getItem(b.Ws.RECENT_EMAILS);return(t?t.split(","):[]).filter(this.isValidEmail.bind(this)).slice(0,3)}pushRecentEmail(t){let e=Array.from(new Set([t,...this.getRecentEmails()])).slice(0,3);b.Ud.setItem(b.Ws.RECENT_EMAILS,e.join(","))}isValidEmail(t){return/^\S+@\S+\.\S+$/u.test(t)}};S.styles=[$],k([(0,a.wk)()],S.prototype,"email",void 0),k([(0,a.wk)()],S.prototype,"address",void 0),k([(0,a.wk)()],S.prototype,"loading",void 0),k([(0,a.wk)()],S.prototype,"appName",void 0),k([(0,a.wk)()],S.prototype,"siwx",void 0),k([(0,a.wk)()],S.prototype,"isRequired",void 0),k([(0,a.wk)()],S.prototype,"recentEmails",void 0),S=k([(0,o.EM)("w3m-data-capture-view")],S)},97734:(t,e,i)=>{i.d(e,{w:()=>a});var r=i(42608);function a(t){return(0,r.M)({...t,state:!0,attribute:!1})}}}]);