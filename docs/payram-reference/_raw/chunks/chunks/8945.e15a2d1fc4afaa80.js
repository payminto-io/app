"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[8945],{15642:(o,t,e)=>{var r=e(88088),i=e(53227);e(19763);var s=e(94895),c=e(69428);let a=(0,r.AH)`
  :host {
    display: inline-flex;
    justify-content: center;
    align-items: center;
    position: relative;
    overflow: hidden;
    background-color: var(--wui-color-gray-glass-020);
    border-radius: var(--local-border-radius);
    border: var(--local-border);
    box-sizing: content-box;
    width: var(--local-size);
    height: var(--local-size);
    min-height: var(--local-size);
    min-width: var(--local-size);
  }

  @supports (background: color-mix(in srgb, white 50%, black)) {
    :host {
      background-color: color-mix(in srgb, var(--local-bg-value) var(--local-bg-mix), transparent);
    }
  }
`;var l=function(o,t,e,r){var i,s=arguments.length,c=s<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,e):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)c=Reflect.decorate(o,t,e,r);else for(var a=o.length-1;a>=0;a--)(i=o[a])&&(c=(s<3?i(c):s>3?i(t,e,c):i(t,e))||c);return s>3&&c&&Object.defineProperty(t,e,c),c};let n=class extends r.WF{constructor(){super(...arguments),this.size="md",this.backgroundColor="accent-100",this.iconColor="accent-100",this.background="transparent",this.border=!1,this.borderColor="wui-color-bg-125",this.icon="copy"}render(){let o=this.iconSize||this.size,t="lg"===this.size,e="xl"===this.size,i="gray"===this.background,s="opaque"===this.background,c="accent-100"===this.backgroundColor&&s||"success-100"===this.backgroundColor&&s||"error-100"===this.backgroundColor&&s||"inverse-100"===this.backgroundColor&&s,a=`var(--wui-color-${this.backgroundColor})`;return c?a=`var(--wui-icon-box-bg-${this.backgroundColor})`:i&&(a=`var(--wui-color-gray-${this.backgroundColor})`),this.style.cssText=`
       --local-bg-value: ${a};
       --local-bg-mix: ${c||i?"100%":t?"12%":"16%"};
       --local-border-radius: var(--wui-border-radius-${t?"xxs":e?"s":"3xl"});
       --local-size: var(--wui-icon-box-size-${this.size});
       --local-border: ${"wui-color-bg-125"===this.borderColor?"2px":"1px"} solid ${this.border?`var(--${this.borderColor})`:"transparent"}
   `,(0,r.qy)` <wui-icon color=${this.iconColor} size=${o} name=${this.icon}></wui-icon> `}};n.styles=[s.W5,s.fD,a],l([(0,i.MZ)()],n.prototype,"size",void 0),l([(0,i.MZ)()],n.prototype,"backgroundColor",void 0),l([(0,i.MZ)()],n.prototype,"iconColor",void 0),l([(0,i.MZ)()],n.prototype,"iconSize",void 0),l([(0,i.MZ)()],n.prototype,"background",void 0),l([(0,i.MZ)({type:Boolean})],n.prototype,"border",void 0),l([(0,i.MZ)()],n.prototype,"borderColor",void 0),l([(0,i.MZ)()],n.prototype,"icon",void 0),n=l([(0,c.E)("wui-icon-box")],n)},37132:(o,t,e)=>{e(15642)},45667:(o,t,e)=>{var r=e(88088),i=e(53227),s=e(40859);e(12936);var c=e(94895),a=e(69428);let l=(0,r.AH)`
  button {
    padding: var(--wui-spacing-4xs) var(--wui-spacing-xxs);
    border-radius: var(--wui-border-radius-3xs);
    background-color: transparent;
    color: var(--wui-color-accent-100);
  }

  button:disabled {
    background-color: transparent;
    color: var(--wui-color-gray-glass-015);
  }

  button:hover {
    background-color: var(--wui-color-gray-glass-005);
  }
`;var n=function(o,t,e,r){var i,s=arguments.length,c=s<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,e):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)c=Reflect.decorate(o,t,e,r);else for(var a=o.length-1;a>=0;a--)(i=o[a])&&(c=(s<3?i(c):s>3?i(t,e,c):i(t,e))||c);return s>3&&c&&Object.defineProperty(t,e,c),c};let d=class extends r.WF{constructor(){super(...arguments),this.tabIdx=void 0,this.disabled=!1,this.color="inherit"}render(){return(0,r.qy)`
      <button ?disabled=${this.disabled} tabindex=${(0,s.J)(this.tabIdx)}>
        <slot name="iconLeft"></slot>
        <wui-text variant="small-600" color=${this.color}>
          <slot></slot>
        </wui-text>
        <slot name="iconRight"></slot>
      </button>
    `}};d.styles=[c.W5,c.fD,l],n([(0,i.MZ)()],d.prototype,"tabIdx",void 0),n([(0,i.MZ)({type:Boolean})],d.prototype,"disabled",void 0),n([(0,i.MZ)()],d.prototype,"color",void 0),d=n([(0,a.E)("wui-link")],d)},68903:(o,t,e)=>{e.r(t),e.d(t,{W3mTransactionsView:()=>c});var r=e(88088),i=e(59306);e(74970),e(58856);let s=(0,r.AH)`
  :host > wui-flex:first-child {
    height: 500px;
    overflow-y: auto;
    overflow-x: hidden;
    scrollbar-width: none;
  }

  :host > wui-flex:first-child::-webkit-scrollbar {
    display: none;
  }
`,c=class extends r.WF{render(){return(0,r.qy)`
      <wui-flex flexDirection="column" .padding=${["0","m","m","m"]} gap="s">
        <w3m-activity-list page="activity"></w3m-activity-list>
      </wui-flex>
    `}};c.styles=s,c=function(o,t,e,r){var i,s=arguments.length,c=s<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,e):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)c=Reflect.decorate(o,t,e,r);else for(var a=o.length-1;a>=0;a--)(i=o[a])&&(c=(s<3?i(c):s>3?i(t,e,c):i(t,e))||c);return s>3&&c&&Object.defineProperty(t,e,c),c}([(0,i.EM)("w3m-transactions-view")],c)},88364:(o,t,e)=>{var r=e(88088),i=e(53227),s=e(94895),c=e(69428);let a=(0,r.AH)`
  :host {
    display: block;
    width: var(--local-width);
    height: var(--local-height);
  }

  :host([data-object-fit='cover']) img {
    object-fit: cover;
    object-position: center center;
  }

  :host([data-object-fit='contain']) img {
    object-fit: contain;
    object-position: center center;
  }

  img {
    display: block;
    width: 100%;
    height: 100%;
    border-radius: inherit;
  }
`;var l=function(o,t,e,r){var i,s=arguments.length,c=s<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,e):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)c=Reflect.decorate(o,t,e,r);else for(var a=o.length-1;a>=0;a--)(i=o[a])&&(c=(s<3?i(c):s>3?i(t,e,c):i(t,e))||c);return s>3&&c&&Object.defineProperty(t,e,c),c};let n=class extends r.WF{constructor(){super(...arguments),this.src="./path/to/image.jpg",this.alt="Image",this.size=void 0,this.objectFit="cover"}render(){return this.objectFit&&(this.dataset.objectFit=this.objectFit),this.style.cssText=`
      --local-width: ${this.size?`var(--wui-icon-size-${this.size});`:"100%"};
      --local-height: ${this.size?`var(--wui-icon-size-${this.size});`:"100%"};
      `,(0,r.qy)`<img src=${this.src} alt=${this.alt} @error=${this.handleImageError} />`}handleImageError(){this.dispatchEvent(new CustomEvent("onLoadError",{bubbles:!0,composed:!0}))}};n.styles=[s.W5,s.ck,a],l([(0,i.MZ)()],n.prototype,"src",void 0),l([(0,i.MZ)()],n.prototype,"alt",void 0),l([(0,i.MZ)()],n.prototype,"size",void 0),l([(0,i.MZ)()],n.prototype,"objectFit",void 0),n=l([(0,c.E)("wui-image")],n)}}]);