"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[7985],{27985:(e,t,r)=>{r.r(t),r.d(t,{W3mWalletReceiveView:()=>$});var i=r(88088),a=r(53227),o=r(40859),s=r(34066),n=r(1024),c=r(44963),l=r(48789),d=r(69148),u=r(33452),h=r(44458),p=r(37056),w=r(59306);r(34664),r(19763),r(88364),r(12936),r(24621);var g=r(94895),v=r(69428);let b=(0,i.AH)`
  button {
    display: flex;
    gap: var(--wui-spacing-xl);
    width: 100%;
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xxs);
    padding: var(--wui-spacing-m) var(--wui-spacing-s);
  }

  wui-text {
    width: 100%;
  }

  wui-flex {
    width: auto;
  }

  .network-icon {
    width: var(--wui-spacing-2l);
    height: var(--wui-spacing-2l);
    border-radius: calc(var(--wui-spacing-2l) / 2);
    overflow: hidden;
    box-shadow:
      0 0 0 3px var(--wui-color-gray-glass-002),
      0 0 0 3px var(--wui-color-modal-bg);
  }
`;var m=function(e,t,r,i){var a,o=arguments.length,s=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,r):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(e,t,r,i);else for(var n=e.length-1;n>=0;n--)(a=e[n])&&(s=(o<3?a(s):o>3?a(t,r,s):a(t,r))||s);return o>3&&s&&Object.defineProperty(t,r,s),s};let f=class extends i.WF{constructor(){super(...arguments),this.networkImages=[""],this.text=""}render(){return(0,i.qy)`
      <button>
        <wui-text variant="small-400" color="fg-200">${this.text}</wui-text>
        <wui-flex gap="3xs" alignItems="center">
          ${this.networksTemplate()}
          <wui-icon name="chevronRight" size="sm" color="fg-200"></wui-icon>
        </wui-flex>
      </button>
    `}networksTemplate(){let e=this.networkImages.slice(0,5);return(0,i.qy)` <wui-flex class="networks">
      ${e?.map(e=>(0,i.qy)` <wui-flex class="network-icon"> <wui-image src=${e}></wui-image> </wui-flex>`)}
    </wui-flex>`}};f.styles=[g.W5,g.fD,b],m([(0,a.MZ)({type:Array})],f.prototype,"networkImages",void 0),m([(0,a.MZ)()],f.prototype,"text",void 0),f=m([(0,v.E)("wui-compatible-network")],f),r(74970),r(92122),r(3658);var y=r(67066);let x=(0,i.AH)`
  wui-compatible-network {
    margin-top: var(--wui-spacing-l);
  }
`;var k=function(e,t,r,i){var a,o=arguments.length,s=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,r):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(e,t,r,i);else for(var n=e.length-1;n>=0;n--)(a=e[n])&&(s=(o<3?a(s):o>3?a(t,r,s):a(t,r))||s);return o>3&&s&&Object.defineProperty(t,r,s),s};let $=class extends i.WF{constructor(){super(),this.unsubscribe=[],this.address=s.U.state.address,this.profileName=s.U.state.profileName,this.network=n.W.state.activeCaipNetwork,this.unsubscribe.push(s.U.subscribe(e=>{e.address?(this.address=e.address,this.profileName=e.profileName):c.P.showError("Account not found")}),n.W.subscribeKey("activeCaipNetwork",e=>{e?.id&&(this.network=e)}))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){if(!this.address)throw Error("w3m-wallet-receive-view: No account provided");let e=l.$.getNetworkImage(this.network);return(0,i.qy)` <wui-flex
      flexDirection="column"
      .padding=${["0","l","l","l"]}
      alignItems="center"
    >
      <wui-chip-button
        data-testid="receive-address-copy-button"
        @click=${this.onCopyClick.bind(this)}
        text=${w.Zv.getTruncateString({string:this.profileName||this.address||"",charsStart:this.profileName?18:4,charsEnd:4*!this.profileName,truncate:this.profileName?"end":"middle"})}
        icon="copy"
        size="sm"
        imageSrc=${e||""}
        variant="gray"
      ></wui-chip-button>
      <wui-flex
        flexDirection="column"
        .padding=${["l","0","0","0"]}
        alignItems="center"
        gap="s"
      >
        <wui-qr-code
          size=${232}
          theme=${d.W.state.themeMode}
          uri=${this.address}
          ?arenaClear=${!0}
          color=${(0,o.J)(d.W.state.themeVariables["--w3m-qr-color"])}
          data-testid="wui-qr-code"
        ></wui-qr-code>
        <wui-text variant="paragraph-500" color="fg-100" align="center">
          Copy your address or scan this QR code
        </wui-text>
      </wui-flex>
      ${this.networkTemplate()}
    </wui-flex>`}networkTemplate(){let e=n.W.getAllRequestedCaipNetworks(),t=n.W.checkIfSmartAccountEnabled(),r=n.W.state.activeCaipNetwork,a=e.filter(e=>e?.chainNamespace===r?.chainNamespace);if((0,u.lj)(r?.chainNamespace)===y.Vl.ACCOUNT_TYPES.SMART_ACCOUNT&&t)return r?(0,i.qy)`<wui-compatible-network
        @click=${this.onReceiveClick.bind(this)}
        text="Only receive assets on this network"
        .networkImages=${[l.$.getNetworkImage(r)??""]}
      ></wui-compatible-network>`:null;let o=(a?.filter(e=>e?.assets?.imageId)?.slice(0,5)).map(l.$.getNetworkImage).filter(Boolean);return(0,i.qy)`<wui-compatible-network
      @click=${this.onReceiveClick.bind(this)}
      text="Only receive assets on these networks"
      .networkImages=${o}
    ></wui-compatible-network>`}onReceiveClick(){h.I.push("WalletCompatibleNetworks")}onCopyClick(){try{this.address&&(p.w.copyToClopboard(this.address),c.P.showSuccess("Address copied"))}catch{c.P.showError("Failed to copy")}}};$.styles=x,k([(0,a.wk)()],$.prototype,"address",void 0),k([(0,a.wk)()],$.prototype,"profileName",void 0),k([(0,a.wk)()],$.prototype,"network",void 0),$=k([(0,w.EM)("w3m-wallet-receive-view")],$)},34664:(e,t,r)=>{var i=r(88088),a=r(53227);r(19763),r(88364),r(12936);var o=r(94895),s=r(69428);let n=(0,i.AH)`
  button {
    border: none;
    border-radius: var(--wui-border-radius-3xl);
  }

  button[data-variant='main'] {
    background-color: var(--wui-color-accent-100);
    color: var(--wui-color-inverse-100);
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-010);
  }

  button[data-variant='accent'] {
    background-color: var(--wui-color-accent-glass-010);
    color: var(--wui-color-accent-100);
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-005);
  }

  button[data-variant='gray'] {
    background-color: transparent;
    color: var(--wui-color-fg-200);
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-010);
  }

  button[data-variant='shade'] {
    background-color: transparent;
    color: var(--wui-color-accent-100);
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-010);
  }

  button[data-size='sm'] {
    height: 32px;
    padding: 0 var(--wui-spacing-s);
  }

  button[data-size='md'] {
    height: 40px;
    padding: 0 var(--wui-spacing-l);
  }

  button[data-size='sm'] > wui-image {
    width: 16px;
    height: 16px;
  }

  button[data-size='md'] > wui-image {
    width: 24px;
    height: 24px;
  }

  button[data-size='sm'] > wui-icon {
    width: 12px;
    height: 12px;
  }

  button[data-size='md'] > wui-icon {
    width: 14px;
    height: 14px;
  }

  wui-image {
    border-radius: var(--wui-border-radius-3xl);
    overflow: hidden;
  }

  button.disabled > wui-icon,
  button.disabled > wui-image {
    filter: grayscale(1);
  }

  button[data-variant='main'] > wui-image {
    box-shadow: inset 0 0 0 1px var(--wui-color-accent-090);
  }

  button[data-variant='shade'] > wui-image,
  button[data-variant='gray'] > wui-image {
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-010);
  }

  @media (hover: hover) and (pointer: fine) {
    button[data-variant='main']:focus-visible {
      background-color: var(--wui-color-accent-090);
    }

    button[data-variant='main']:hover:enabled {
      background-color: var(--wui-color-accent-090);
    }

    button[data-variant='main']:active:enabled {
      background-color: var(--wui-color-accent-080);
    }

    button[data-variant='accent']:hover:enabled {
      background-color: var(--wui-color-accent-glass-015);
    }

    button[data-variant='accent']:active:enabled {
      background-color: var(--wui-color-accent-glass-020);
    }

    button[data-variant='shade']:focus-visible,
    button[data-variant='gray']:focus-visible,
    button[data-variant='shade']:hover,
    button[data-variant='gray']:hover {
      background-color: var(--wui-color-gray-glass-002);
    }

    button[data-variant='gray']:active,
    button[data-variant='shade']:active {
      background-color: var(--wui-color-gray-glass-005);
    }
  }

  button.disabled {
    color: var(--wui-color-gray-glass-020);
    background-color: var(--wui-color-gray-glass-002);
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-002);
    pointer-events: none;
  }
`;var c=function(e,t,r,i){var a,o=arguments.length,s=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,r):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(e,t,r,i);else for(var n=e.length-1;n>=0;n--)(a=e[n])&&(s=(o<3?a(s):o>3?a(t,r,s):a(t,r))||s);return o>3&&s&&Object.defineProperty(t,r,s),s};let l=class extends i.WF{constructor(){super(...arguments),this.variant="accent",this.imageSrc="",this.disabled=!1,this.icon="externalLink",this.size="md",this.text=""}render(){let e="sm"===this.size?"small-600":"paragraph-600";return(0,i.qy)`
      <button
        class=${this.disabled?"disabled":""}
        data-variant=${this.variant}
        data-size=${this.size}
        ?disabled=${this.disabled}
      >
        ${this.imageSrc?(0,i.qy)`<wui-image src=${this.imageSrc}></wui-image>`:null}
        <wui-text variant=${e} color="inherit"> ${this.text} </wui-text>
        ${this.icon?(0,i.qy)`<wui-icon name=${this.icon} color="inherit" size="inherit"></wui-icon>`:null}
      </button>
    `}};l.styles=[o.W5,o.fD,n],c([(0,a.MZ)()],l.prototype,"variant",void 0),c([(0,a.MZ)()],l.prototype,"imageSrc",void 0),c([(0,a.MZ)({type:Boolean})],l.prototype,"disabled",void 0),c([(0,a.MZ)()],l.prototype,"icon",void 0),c([(0,a.MZ)()],l.prototype,"size",void 0),c([(0,a.MZ)()],l.prototype,"text",void 0),l=c([(0,s.E)("wui-chip-button")],l)},88364:(e,t,r)=>{var i=r(88088),a=r(53227),o=r(94895),s=r(69428);let n=(0,i.AH)`
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
`;var c=function(e,t,r,i){var a,o=arguments.length,s=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,r):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(e,t,r,i);else for(var n=e.length-1;n>=0;n--)(a=e[n])&&(s=(o<3?a(s):o>3?a(t,r,s):a(t,r))||s);return o>3&&s&&Object.defineProperty(t,r,s),s};let l=class extends i.WF{constructor(){super(...arguments),this.src="./path/to/image.jpg",this.alt="Image",this.size=void 0,this.objectFit="cover"}render(){return this.objectFit&&(this.dataset.objectFit=this.objectFit),this.style.cssText=`
      --local-width: ${this.size?`var(--wui-icon-size-${this.size});`:"100%"};
      --local-height: ${this.size?`var(--wui-icon-size-${this.size});`:"100%"};
      `,(0,i.qy)`<img src=${this.src} alt=${this.alt} @error=${this.handleImageError} />`}handleImageError(){this.dispatchEvent(new CustomEvent("onLoadError",{bubbles:!0,composed:!0}))}};l.styles=[o.W5,o.ck,n],c([(0,a.MZ)()],l.prototype,"src",void 0),c([(0,a.MZ)()],l.prototype,"alt",void 0),c([(0,a.MZ)()],l.prototype,"size",void 0),c([(0,a.MZ)()],l.prototype,"objectFit",void 0),l=c([(0,s.E)("wui-image")],l)},92122:(e,t,r)=>{var i=r(88088),a=r(53227);r(19763),r(88364);var o=r(768);function s(e,t,r){return e!==t&&(e-t<0?t-e:e-t)<=r+.1}let n={generate({uri:e,size:t,logoSize:r,dotColor:a="#141414"}){let n,c,l=[],d=(c=Math.sqrt((n=Array.prototype.slice.call(o.create(e,{errorCorrectionLevel:"Q"}).modules.data,0)).length),n.reduce((e,t,r)=>(r%c==0?e.push([t]):e[e.length-1].push(t))&&e,[])),u=t/d.length,h=[{x:0,y:0},{x:1,y:0},{x:0,y:1}];h.forEach(({x:e,y:t})=>{let r=(d.length-7)*u*e,o=(d.length-7)*u*t;for(let e=0;e<h.length;e+=1){let t=u*(7-2*e);l.push((0,i.JW)`
            <rect
              fill=${2===e?a:"transparent"}
              width=${0===e?t-5:t}
              rx= ${0===e?(t-5)*.45:.45*t}
              ry= ${0===e?(t-5)*.45:.45*t}
              stroke=${a}
              stroke-width=${5*(0===e)}
              height=${0===e?t-5:t}
              x= ${0===e?o+u*e+2.5:o+u*e}
              y= ${0===e?r+u*e+2.5:r+u*e}
            />
          `)}});let p=Math.floor((r+25)/u),w=d.length/2-p/2,g=d.length/2+p/2-1,v=[];d.forEach((e,t)=>{e.forEach((e,r)=>{!d[t][r]||t<7&&r<7||t>d.length-8&&r<7||t<7&&r>d.length-8||t>w&&t<g&&r>w&&r<g||v.push([t*u+u/2,r*u+u/2])})});let b={};return v.forEach(([e,t])=>{b[e]?b[e]?.push(t):b[e]=[t]}),Object.entries(b).map(([e,t])=>{let r=t.filter(e=>t.every(t=>!s(e,t,u)));return[Number(e),r]}).forEach(([e,t])=>{t.forEach(t=>{l.push((0,i.JW)`<circle cx=${e} cy=${t} fill=${a} r=${u/2.5} />`)})}),Object.entries(b).filter(([e,t])=>t.length>1).map(([e,t])=>{let r=t.filter(e=>t.some(t=>s(e,t,u)));return[Number(e),r]}).map(([e,t])=>{t.sort((e,t)=>e<t?-1:1);let r=[];for(let e of t){let t=r.find(t=>t.some(t=>s(e,t,u)));t?t.push(e):r.push([e])}return[e,r.map(e=>[e[0],e[e.length-1]])]}).forEach(([e,t])=>{t.forEach(([t,r])=>{l.push((0,i.JW)`
              <line
                x1=${e}
                x2=${e}
                y1=${t}
                y2=${r}
                stroke=${a}
                stroke-width=${u/1.25}
                stroke-linecap="round"
              />
            `)})}),l}};var c=r(94895),l=r(69428);let d=(0,i.AH)`
  :host {
    position: relative;
    user-select: none;
    display: block;
    overflow: hidden;
    aspect-ratio: 1 / 1;
    width: var(--local-size);
  }

  :host([data-theme='dark']) {
    border-radius: clamp(0px, var(--wui-border-radius-l), 40px);
    background-color: var(--wui-color-inverse-100);
    padding: var(--wui-spacing-l);
  }

  :host([data-theme='light']) {
    box-shadow: 0 0 0 1px var(--wui-color-bg-125);
    background-color: var(--wui-color-bg-125);
  }

  :host([data-clear='true']) > wui-icon {
    display: none;
  }

  svg:first-child,
  wui-image,
  wui-icon {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translateY(-50%) translateX(-50%);
  }

  wui-image {
    width: 25%;
    height: 25%;
    border-radius: var(--wui-border-radius-xs);
  }

  wui-icon {
    width: 100%;
    height: 100%;
    color: var(--local-icon-color) !important;
    transform: translateY(-50%) translateX(-50%) scale(0.25);
  }
`;var u=function(e,t,r,i){var a,o=arguments.length,s=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,r):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(e,t,r,i);else for(var n=e.length-1;n>=0;n--)(a=e[n])&&(s=(o<3?a(s):o>3?a(t,r,s):a(t,r))||s);return o>3&&s&&Object.defineProperty(t,r,s),s};let h=class extends i.WF{constructor(){super(...arguments),this.uri="",this.size=0,this.theme="dark",this.imageSrc=void 0,this.alt=void 0,this.arenaClear=void 0,this.farcaster=void 0}render(){return this.dataset.theme=this.theme,this.dataset.clear=String(this.arenaClear),this.style.cssText=`
     --local-size: ${this.size}px;
     --local-icon-color: ${this.color??"#3396ff"}
    `,(0,i.qy)`${this.templateVisual()} ${this.templateSvg()}`}templateSvg(){let e="light"===this.theme?this.size:this.size-32;return(0,i.JW)`
      <svg height=${e} width=${e}>
        ${n.generate({uri:this.uri,size:e,logoSize:this.arenaClear?0:e/4,dotColor:this.color})}
      </svg>
    `}templateVisual(){return this.imageSrc?(0,i.qy)`<wui-image src=${this.imageSrc} alt=${this.alt??"logo"}></wui-image>`:this.farcaster?(0,i.qy)`<wui-icon
        class="farcaster"
        size="inherit"
        color="inherit"
        name="farcaster"
      ></wui-icon>`:(0,i.qy)`<wui-icon size="inherit" color="inherit" name="walletConnect"></wui-icon>`}};h.styles=[c.W5,d],u([(0,a.MZ)()],h.prototype,"uri",void 0),u([(0,a.MZ)({type:Number})],h.prototype,"size",void 0),u([(0,a.MZ)()],h.prototype,"theme",void 0),u([(0,a.MZ)()],h.prototype,"imageSrc",void 0),u([(0,a.MZ)()],h.prototype,"alt",void 0),u([(0,a.MZ)()],h.prototype,"color",void 0),u([(0,a.MZ)({type:Boolean})],h.prototype,"arenaClear",void 0),u([(0,a.MZ)({type:Boolean})],h.prototype,"farcaster",void 0),h=u([(0,l.E)("wui-qr-code")],h)}}]);