"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[9677],{12933:(e,t,a)=>{a(80124)},15642:(e,t,a)=>{var i=a(88088),r=a(53227);a(19763);var o=a(94895),n=a(69428);let s=(0,i.AH)`
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
`;var c=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let l=class extends i.WF{constructor(){super(...arguments),this.size="md",this.backgroundColor="accent-100",this.iconColor="accent-100",this.background="transparent",this.border=!1,this.borderColor="wui-color-bg-125",this.icon="copy"}render(){let e=this.iconSize||this.size,t="lg"===this.size,a="xl"===this.size,r="gray"===this.background,o="opaque"===this.background,n="accent-100"===this.backgroundColor&&o||"success-100"===this.backgroundColor&&o||"error-100"===this.backgroundColor&&o||"inverse-100"===this.backgroundColor&&o,s=`var(--wui-color-${this.backgroundColor})`;return n?s=`var(--wui-icon-box-bg-${this.backgroundColor})`:r&&(s=`var(--wui-color-gray-${this.backgroundColor})`),this.style.cssText=`
       --local-bg-value: ${s};
       --local-bg-mix: ${n||r?"100%":t?"12%":"16%"};
       --local-border-radius: var(--wui-border-radius-${t?"xxs":a?"s":"3xl"});
       --local-size: var(--wui-icon-box-size-${this.size});
       --local-border: ${"wui-color-bg-125"===this.borderColor?"2px":"1px"} solid ${this.border?`var(--${this.borderColor})`:"transparent"}
   `,(0,i.qy)` <wui-icon color=${this.iconColor} size=${e} name=${this.icon}></wui-icon> `}};l.styles=[o.W5,o.fD,s],c([(0,r.MZ)()],l.prototype,"size",void 0),c([(0,r.MZ)()],l.prototype,"backgroundColor",void 0),c([(0,r.MZ)()],l.prototype,"iconColor",void 0),c([(0,r.MZ)()],l.prototype,"iconSize",void 0),c([(0,r.MZ)()],l.prototype,"background",void 0),c([(0,r.MZ)({type:Boolean})],l.prototype,"border",void 0),c([(0,r.MZ)()],l.prototype,"borderColor",void 0),c([(0,r.MZ)()],l.prototype,"icon",void 0),l=c([(0,n.E)("wui-icon-box")],l)},15762:(e,t,a)=>{var i=a(88088),r=a(53227),o=a(94895),n=a(69428);let s=(0,i.AH)`
  :host {
    display: block;
    width: var(--wui-box-size-md);
    height: var(--wui-box-size-md);
  }

  svg {
    width: var(--wui-box-size-md);
    height: var(--wui-box-size-md);
  }

  rect {
    fill: none;
    stroke: var(--wui-color-accent-100);
    stroke-width: 4px;
    stroke-linecap: round;
    animation: dash 1s linear infinite;
  }

  @keyframes dash {
    to {
      stroke-dashoffset: 0px;
    }
  }
`;var c=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let l=class extends i.WF{constructor(){super(...arguments),this.radius=36}render(){return this.svgLoaderTemplate()}svgLoaderTemplate(){let e=this.radius>50?50:this.radius,t=36-e;return(0,i.qy)`
      <svg viewBox="0 0 110 110" width="110" height="110">
        <rect
          x="2"
          y="2"
          width="106"
          height="106"
          rx=${e}
          stroke-dasharray="${116+t} ${245+t}"
          stroke-dashoffset=${360+1.75*t}
        />
      </svg>
    `}};l.styles=[o.W5,s],c([(0,r.MZ)({type:Number})],l.prototype,"radius",void 0),l=c([(0,n.E)("wui-loading-thumbnail")],l)},18128:(e,t,a)=>{a(19690)},19690:(e,t,a)=>{var i=a(88088),r=a(53227);a(19763),a(88364),a(24621);var o=a(94895),n=a(69428);a(15642);let s=(0,i.AH)`
  :host {
    position: relative;
    background-color: var(--wui-color-gray-glass-002);
    display: flex;
    justify-content: center;
    align-items: center;
    width: var(--local-size);
    height: var(--local-size);
    border-radius: inherit;
    border-radius: var(--local-border-radius);
  }

  :host > wui-flex {
    overflow: hidden;
    border-radius: inherit;
    border-radius: var(--local-border-radius);
  }

  :host::after {
    content: '';
    position: absolute;
    top: 0;
    bottom: 0;
    left: 0;
    right: 0;
    border-radius: inherit;
    border: 1px solid var(--wui-color-gray-glass-010);
    pointer-events: none;
  }

  :host([name='Extension'])::after {
    border: 1px solid var(--wui-color-accent-glass-010);
  }

  :host([data-wallet-icon='allWallets']) {
    background-color: var(--wui-all-wallets-bg-100);
  }

  :host([data-wallet-icon='allWallets'])::after {
    border: 1px solid var(--wui-color-accent-glass-010);
  }

  wui-icon[data-parent-size='inherit'] {
    width: 75%;
    height: 75%;
    align-items: center;
  }

  wui-icon[data-parent-size='sm'] {
    width: 18px;
    height: 18px;
  }

  wui-icon[data-parent-size='md'] {
    width: 24px;
    height: 24px;
  }

  wui-icon[data-parent-size='lg'] {
    width: 42px;
    height: 42px;
  }

  wui-icon[data-parent-size='full'] {
    width: 100%;
    height: 100%;
  }

  :host > wui-icon-box {
    position: absolute;
    overflow: hidden;
    right: -1px;
    bottom: -2px;
    z-index: 1;
    border: 2px solid var(--wui-color-bg-150, #1e1f1f);
    padding: 1px;
  }
`;var c=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let l=class extends i.WF{constructor(){super(...arguments),this.size="md",this.name="",this.installed=!1,this.badgeSize="xs"}render(){let e="xxs";return e="lg"===this.size?"m":"md"===this.size?"xs":"xxs",this.style.cssText=`
       --local-border-radius: var(--wui-border-radius-${e});
       --local-size: var(--wui-wallet-image-size-${this.size});
   `,this.walletIcon&&(this.dataset.walletIcon=this.walletIcon),(0,i.qy)`
      <wui-flex justifyContent="center" alignItems="center"> ${this.templateVisual()} </wui-flex>
    `}templateVisual(){return this.imageSrc?(0,i.qy)`<wui-image src=${this.imageSrc} alt=${this.name}></wui-image>`:this.walletIcon?(0,i.qy)`<wui-icon
        data-parent-size="md"
        size="md"
        color="inherit"
        name=${this.walletIcon}
      ></wui-icon>`:(0,i.qy)`<wui-icon
      data-parent-size=${this.size}
      size="inherit"
      color="inherit"
      name="walletPlaceholder"
    ></wui-icon>`}};l.styles=[o.fD,o.W5,s],c([(0,r.MZ)()],l.prototype,"size",void 0),c([(0,r.MZ)()],l.prototype,"name",void 0),c([(0,r.MZ)()],l.prototype,"imageSrc",void 0),c([(0,r.MZ)()],l.prototype,"walletIcon",void 0),c([(0,r.MZ)({type:Boolean})],l.prototype,"installed",void 0),c([(0,r.MZ)()],l.prototype,"badgeSize",void 0),l=c([(0,n.E)("wui-wallet-image")],l)},20903:(e,t,a)=>{var i=a(88088),r=a(53227);a(19763);var o=a(94895),n=a(69428);let s=(0,i.AH)`
  button {
    border-radius: var(--local-border-radius);
    color: var(--wui-color-fg-100);
    padding: var(--local-padding);
  }

  @media (max-width: 700px) {
    :host(:not([size='sm'])) button {
      padding: var(--wui-spacing-s);
    }
  }

  button > wui-icon {
    pointer-events: none;
  }

  button:disabled > wui-icon {
    color: var(--wui-color-bg-300) !important;
  }

  button:disabled {
    background-color: transparent;
  }

  button:hover:not(:disabled) {
    background-color: var(--wui-color-accent-glass-015);
  }

  button:focus-visible:not(:disabled) {
    background-color: var(--wui-color-accent-glass-015);
    box-shadow:
      inset 0 0 0 1px var(--wui-color-accent-100),
      0 0 0 4px var(--wui-color-accent-glass-020);
  }
`;var c=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let l=class extends i.WF{constructor(){super(...arguments),this.size="md",this.disabled=!1,this.icon="copy",this.iconColor="inherit"}render(){this.dataset.size=this.size;let e="",t="";switch(this.size){case"lg":e="--wui-border-radius-xs",t="--wui-spacing-1xs";break;case"sm":e="--wui-border-radius-3xs",t="--wui-spacing-xxs";break;default:e="--wui-border-radius-xxs",t="--wui-spacing-2xs"}return this.style.cssText=`
    --local-border-radius: var(${e});
    --local-padding: var(${t});
    `,(0,i.qy)`
      <button ?disabled=${this.disabled}>
        <wui-icon color=${this.iconColor} size=${this.size} name=${this.icon}></wui-icon>
      </button>
    `}};l.styles=[o.W5,o.fD,o.ck,s],c([(0,r.MZ)()],l.prototype,"size",void 0),c([(0,r.MZ)({type:Boolean})],l.prototype,"disabled",void 0),c([(0,r.MZ)()],l.prototype,"icon",void 0),c([(0,r.MZ)()],l.prototype,"iconColor",void 0),l=c([(0,n.E)("wui-icon-link")],l)},31320:(e,t,a)=>{a.d(t,{a:()=>r});var i=a(88088);let r=(0,i.JW)`<svg  viewBox="0 0 48 54" fill="none">
  <path
    d="M43.4605 10.7248L28.0485 1.61089C25.5438 0.129705 22.4562 0.129705 19.9515 1.61088L4.53951 10.7248C2.03626 12.2051 0.5 14.9365 0.5 17.886V36.1139C0.5 39.0635 2.03626 41.7949 4.53951 43.2752L19.9515 52.3891C22.4562 53.8703 25.5438 53.8703 28.0485 52.3891L43.4605 43.2752C45.9637 41.7949 47.5 39.0635 47.5 36.114V17.8861C47.5 14.9365 45.9637 12.2051 43.4605 10.7248Z"
  />
</svg>`},35787:(e,t,a)=>{a(20903)},48342:(e,t,a)=>{a(88364)},52589:(e,t,a)=>{var i=a(88088),r=a(53227);a(12936);var o=a(94895),n=a(69428);let s=(0,i.AH)`
  :host {
    position: relative;
    display: flex;
    width: 100%;
    height: 1px;
    background-color: var(--wui-color-gray-glass-005);
    justify-content: center;
    align-items: center;
  }

  :host > wui-text {
    position: absolute;
    padding: 0px 10px;
    background-color: var(--wui-color-modal-bg);
    transition: background-color var(--wui-duration-lg) var(--wui-ease-out-power-1);
    will-change: background-color;
  }
`;var c=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let l=class extends i.WF{constructor(){super(...arguments),this.text=""}render(){return(0,i.qy)`${this.template()}`}template(){return this.text?(0,i.qy)`<wui-text variant="small-500" color="fg-200">${this.text}</wui-text>`:null}};l.styles=[o.W5,s],c([(0,r.MZ)()],l.prototype,"text",void 0),l=c([(0,n.E)("wui-separator")],l)},59677:(e,t,a)=>{a.r(t),a.d(t,{W3mPayLoadingView:()=>er,W3mPayView:()=>X,arbitrumUSDC:()=>ev,arbitrumUSDT:()=>ek,baseETH:()=>ep,baseSepoliaETH:()=>eg,baseUSDC:()=>eh,ethereumUSDC:()=>ew,ethereumUSDT:()=>ef,getExchanges:()=>es,getIsPaymentInProgress:()=>eu,getPayError:()=>el,getPayResult:()=>ec,openPay:()=>eo,optimismUSDC:()=>ey,optimismUSDT:()=>ex,pay:()=>en,polygonUSDC:()=>em,polygonUSDT:()=>eP,solanaSOL:()=>eI,solanaUSDC:()=>eb,solanaUSDT:()=>eE});var i=a(88088),r=a(53227),o=a(40859),n=a(34066),s=a(1024),c=a(24568),l=a(37056),u=a(44963),d=a(70106),p=a(59306);a(66449),a(74970),a(94548),a(92730),a(35787),a(48342),a(61327),a(88844),a(12933),a(52589),a(3658),a(18128);var h=a(45430),g=a(3125),w=a(34386),y=a(30840),v=a(43242),m=a(44458),b=a(53100);let f="INVALID_PAYMENT_CONFIG",x="INVALID_RECIPIENT",k="INVALID_ASSET",P="INVALID_AMOUNT",E="UNABLE_TO_INITIATE_PAYMENT",I="INVALID_CHAIN_NAMESPACE",S="GENERIC_PAYMENT_ERROR",C="UNABLE_TO_GET_EXCHANGES",A="ASSET_NOT_SUPPORTED",z="UNABLE_TO_GET_PAY_URL",$="UNABLE_TO_GET_BUY_STATUS",N={[f]:"Invalid payment configuration",[x]:"Invalid recipient address",[k]:"Invalid asset specified",[P]:"Invalid payment amount",UNKNOWN_ERROR:"Unknown payment error occurred",[E]:"Unable to initiate payment",[I]:"Invalid chain namespace",[S]:"Unable to process payment",[C]:"Unable to get exchanges",[A]:"Asset not supported by the selected exchange",[z]:"Unable to get payment URL",[$]:"Unable to get buy status"};class T extends Error{get message(){return N[this.code]}constructor(e,t){super(N[e]),this.name="AppKitPayError",this.code=e,this.details=t,Error.captureStackTrace&&Error.captureStackTrace(this,T)}}var R=a(19392);class M extends Error{}async function D(e,t){let a,i=(a=R.H.getSnapshot().projectId,`https://rpc.walletconnect.org/v1/json-rpc?projectId=${a}`),{sdkType:r,sdkVersion:o,projectId:n}=R.H.getSnapshot(),s={jsonrpc:"2.0",id:1,method:e,params:{...t||{},st:r,sv:o,projectId:n}},c=await fetch(i,{method:"POST",body:JSON.stringify(s),headers:{"Content-Type":"application/json"}}),l=await c.json();if(l.error)throw new M(l.error.message);return l}async function U(e){return(await D("reown_getExchanges",e)).result}async function W(e){return(await D("reown_getExchangePayUrl",e)).result}async function O(e){return(await D("reown_getExchangeBuyStatus",e)).result}let j=["eip155","solana"],q={eip155:{native:{assetNamespace:"slip44",assetReference:"60"},defaultTokenNamespace:"erc20"},solana:{native:{assetNamespace:"slip44",assetReference:"501"},defaultTokenNamespace:"token"}};function _(e,t){let{chainNamespace:a,chainId:i}=y.C.parseCaipNetworkId(e),r=q[a];if(!r)throw Error(`Unsupported chain namespace for CAIP-19 formatting: ${a}`);let o=r.native.assetNamespace,n=r.native.assetReference;"native"!==t&&(o=r.defaultTokenNamespace,n=t);let s=`${a}:${i}`;return`${s}/${o}:${n}`}var Z=a(92087);async function L(e){let{paymentAssetNetwork:t,activeCaipNetwork:a,approvedCaipNetworkIds:i,requestedCaipNetworks:r}=e,o=l.w.sortRequestedNetworks(i,r).find(e=>e.caipNetworkId===t);if(!o)throw new T(f);if(o.caipNetworkId===a.caipNetworkId)return;let n=s.W.getNetworkProp("supportsAllNetworks",o.chainNamespace);if(!(i?.includes(o.caipNetworkId)||n))throw new T(f);try{await s.W.switchActiveNetwork(o)}catch(e){throw new T(S,e)}}async function F(e,t,a){if(t!==w.o.CHAIN.EVM)throw new T(I);if(!a.fromAddress)throw new T(f,"fromAddress is required for native EVM payments.");let i="string"==typeof a.amount?parseFloat(a.amount):a.amount;if(isNaN(i))throw new T(f);let r=e.metadata?.decimals??18,o=d.x.parseUnits(i.toString(),r);if("bigint"!=typeof o)throw new T(S);return await d.x.sendTransaction({chainNamespace:t,to:a.recipient,address:a.fromAddress,value:o,data:"0x"})??void 0}async function B(e,t){if(!t.fromAddress)throw new T(f,"fromAddress is required for ERC20 EVM payments.");let a=e.asset,i=t.recipient,r=Number(e.metadata.decimals),o=d.x.parseUnits(t.amount.toString(),r);if(void 0===o)throw new T(S);return await d.x.writeContract({fromAddress:t.fromAddress,tokenAddress:a,args:[i,o],method:"transfer",abi:Z.v.getERC20Abi(a),chainNamespace:w.o.CHAIN.EVM})??void 0}async function V(e,t){if(e!==w.o.CHAIN.SOLANA)throw new T(I);if(!t.fromAddress)throw new T(f,"fromAddress is required for Solana payments.");let a="string"==typeof t.amount?parseFloat(t.amount):t.amount;if(isNaN(a)||a<=0)throw new T(f,"Invalid payment amount.");try{if(!b.A.getProvider(e))throw new T(S,"No Solana provider available.");let i=await d.x.sendTransaction({chainNamespace:w.o.CHAIN.SOLANA,to:t.recipient,value:a,tokenMint:t.tokenMint});if(!i)throw new T(S,"Transaction failed.");return i}catch(e){if(e instanceof T)throw e;throw new T(S,`Solana payment failed: ${e}`)}}let H="unknown",Y=(0,h.BX)({paymentAsset:{network:"eip155:1",asset:"0x0",metadata:{name:"0x0",symbol:"0x0",decimals:0}},recipient:"0x0",amount:0,isConfigured:!1,error:null,isPaymentInProgress:!1,exchanges:[],isLoading:!1,openInNewTab:!0,redirectUrl:void 0,payWithExchange:void 0,currentPayment:void 0,analyticsSet:!1,paymentId:void 0}),G={state:Y,subscribe:e=>(0,h.B1)(Y,()=>e(Y)),subscribeKey:(e,t)=>(0,g.u$)(Y,e,t),async handleOpenPay(e){this.resetState(),this.setPaymentConfig(e),this.subscribeEvents(),this.initializeAnalytics(),Y.isConfigured=!0,v.E.sendEvent({type:"track",event:"PAY_MODAL_OPEN",properties:{exchanges:Y.exchanges,configuration:{network:Y.paymentAsset.network,asset:Y.paymentAsset.asset,recipient:Y.recipient,amount:Y.amount}}}),await c.W.open({view:"Pay"})},resetState(){Y.paymentAsset={network:"eip155:1",asset:"0x0",metadata:{name:"0x0",symbol:"0x0",decimals:0}},Y.recipient="0x0",Y.amount=0,Y.isConfigured=!1,Y.error=null,Y.isPaymentInProgress=!1,Y.isLoading=!1,Y.currentPayment=void 0},setPaymentConfig(e){if(!e.paymentAsset)throw new T(f);try{Y.paymentAsset=e.paymentAsset,Y.recipient=e.recipient,Y.amount=e.amount,Y.openInNewTab=e.openInNewTab??!0,Y.redirectUrl=e.redirectUrl,Y.payWithExchange=e.payWithExchange,Y.error=null}catch(e){throw new T(f,e.message)}},getPaymentAsset:()=>Y.paymentAsset,getExchanges:()=>Y.exchanges,async fetchExchanges(){try{Y.isLoading=!0;let e=await U({page:0,asset:_(Y.paymentAsset.network,Y.paymentAsset.asset),amount:Y.amount.toString()});Y.exchanges=e.exchanges.slice(0,2)}catch(e){throw u.P.showError(N.UNABLE_TO_GET_EXCHANGES),new T(C)}finally{Y.isLoading=!1}},async getAvailableExchanges(e){try{let t=e?.asset&&e?.network?_(e.network,e.asset):void 0;return await U({page:e?.page??0,asset:t,amount:e?.amount?.toString()})}catch(e){throw new T(C)}},async getPayUrl(e,t,a=!1){try{let i=Number(t.amount),r=await W({exchangeId:e,asset:_(t.network,t.asset),amount:i.toString(),recipient:`${t.network}:${t.recipient}`});return v.E.sendEvent({type:"track",event:"PAY_EXCHANGE_SELECTED",properties:{source:"pay",exchange:{id:e},configuration:{network:t.network,asset:t.asset,recipient:t.recipient,amount:i},currentPayment:{type:"exchange",exchangeId:e},headless:a}}),a&&(this.initiatePayment(),v.E.sendEvent({type:"track",event:"PAY_INITIATED",properties:{source:"pay",paymentId:Y.paymentId||H,configuration:{network:t.network,asset:t.asset,recipient:t.recipient,amount:i},currentPayment:{type:"exchange",exchangeId:e}}})),r}catch(e){if(e instanceof Error&&e.message.includes("is not supported"))throw new T(A);throw Error(e.message)}},async openPayUrl(e,t,a=!1){try{let i=await this.getPayUrl(e.exchangeId,t,a);if(!i)throw new T(z);let r=e.openInNewTab??!0;return l.w.openHref(i.url,r?"_blank":"_self"),i}catch(e){throw e instanceof T?Y.error=e.message:Y.error=N.GENERIC_PAYMENT_ERROR,new T(z)}},subscribeEvents(){Y.isConfigured||(d.x.subscribeKey("connections",e=>{e.size>0&&this.handlePayment()}),n.U.subscribeKey("caipAddress",e=>{let t=d.x.hasAnyConnection(w.o.CONNECTOR_ID.WALLET_CONNECT);e&&(t?setTimeout(()=>{this.handlePayment()},100):this.handlePayment())}))},async handlePayment(){Y.currentPayment={type:"wallet",status:"IN_PROGRESS"};let e=n.U.state.caipAddress;if(!e)return;let{chainId:t,address:a}=y.C.parseCaipAddress(e),i=s.W.state.activeChain;if(!a||!t||!i||!b.A.getProvider(i))return;let r=s.W.state.activeCaipNetwork;if(r&&!Y.isPaymentInProgress)try{this.initiatePayment();let e=s.W.getAllRequestedCaipNetworks(),t=s.W.getAllApprovedCaipNetworkIds();switch(await L({paymentAssetNetwork:Y.paymentAsset.network,activeCaipNetwork:r,approvedCaipNetworkIds:t,requestedCaipNetworks:e}),await c.W.open({view:"PayLoading"}),i){case w.o.CHAIN.EVM:"native"===Y.paymentAsset.asset&&(Y.currentPayment.result=await F(Y.paymentAsset,i,{recipient:Y.recipient,amount:Y.amount,fromAddress:a})),Y.paymentAsset.asset.startsWith("0x")&&(Y.currentPayment.result=await B(Y.paymentAsset,{recipient:Y.recipient,amount:Y.amount,fromAddress:a})),Y.currentPayment.status="SUCCESS";break;case w.o.CHAIN.SOLANA:Y.currentPayment.result=await V(i,{recipient:Y.recipient,amount:Y.amount,fromAddress:a,tokenMint:"native"===Y.paymentAsset.asset?void 0:Y.paymentAsset.asset}),Y.currentPayment.status="SUCCESS";break;default:throw new T(I)}}catch(e){e instanceof T?Y.error=e.message:Y.error=N.GENERIC_PAYMENT_ERROR,Y.currentPayment.status="FAILED",u.P.showError(Y.error)}finally{Y.isPaymentInProgress=!1}},getExchangeById:e=>Y.exchanges.find(t=>t.id===e),validatePayConfig(e){let{paymentAsset:t,recipient:a,amount:i}=e;if(!t)throw new T(f);if(!a)throw new T(x);if(!t.asset)throw new T(k);if(null==i||i<=0)throw new T(P)},handlePayWithWallet(){let e=n.U.state.caipAddress;if(!e)return void m.I.push("Connect");let{chainId:t,address:a}=y.C.parseCaipAddress(e),i=s.W.state.activeChain;a&&t&&i?this.handlePayment():m.I.push("Connect")},async handlePayWithExchange(e){try{Y.currentPayment={type:"exchange",exchangeId:e};let{network:t,asset:a}=Y.paymentAsset,i={network:t,asset:a,amount:Y.amount,recipient:Y.recipient},r=await this.getPayUrl(e,i);if(!r)throw new T(E);return Y.currentPayment.sessionId=r.sessionId,Y.currentPayment.status="IN_PROGRESS",Y.currentPayment.exchangeId=e,this.initiatePayment(),{url:r.url,openInNewTab:Y.openInNewTab}}catch(e){return e instanceof T?Y.error=e.message:Y.error=N.GENERIC_PAYMENT_ERROR,Y.isPaymentInProgress=!1,u.P.showError(Y.error),null}},async getBuyStatus(e,t){try{let a=await O({sessionId:t,exchangeId:e});return("SUCCESS"===a.status||"FAILED"===a.status)&&v.E.sendEvent({type:"track",event:"SUCCESS"===a.status?"PAY_SUCCESS":"PAY_ERROR",properties:{source:"pay",paymentId:Y.paymentId||H,configuration:{network:Y.paymentAsset.network,asset:Y.paymentAsset.asset,recipient:Y.recipient,amount:Y.amount},currentPayment:{type:"exchange",exchangeId:Y.currentPayment?.exchangeId,sessionId:Y.currentPayment?.sessionId,result:a.txHash}}}),a}catch(e){throw new T($)}},async updateBuyStatus(e,t){try{let a=await this.getBuyStatus(e,t);Y.currentPayment&&(Y.currentPayment.status=a.status,Y.currentPayment.result=a.txHash),("SUCCESS"===a.status||"FAILED"===a.status)&&(Y.isPaymentInProgress=!1)}catch(e){throw new T($)}},initiatePayment(){Y.isPaymentInProgress=!0,Y.paymentId=crypto.randomUUID()},initializeAnalytics(){Y.analyticsSet||(Y.analyticsSet=!0,this.subscribeKey("isPaymentInProgress",e=>{if(Y.currentPayment?.status&&"UNKNOWN"!==Y.currentPayment.status){let e={IN_PROGRESS:"PAY_INITIATED",SUCCESS:"PAY_SUCCESS",FAILED:"PAY_ERROR"}[Y.currentPayment.status];v.E.sendEvent({type:"track",event:e,properties:{source:"pay",paymentId:Y.paymentId||H,configuration:{network:Y.paymentAsset.network,asset:Y.paymentAsset.asset,recipient:Y.recipient,amount:Y.amount},currentPayment:{type:Y.currentPayment.type,exchangeId:Y.currentPayment.exchangeId,sessionId:Y.currentPayment.sessionId,result:Y.currentPayment.result}}})}}))}},K=(0,i.AH)`
  wui-separator {
    margin: var(--wui-spacing-m) calc(var(--wui-spacing-m) * -1) var(--wui-spacing-xs)
      calc(var(--wui-spacing-m) * -1);
    width: calc(100% + var(--wui-spacing-s) * 2);
  }

  .token-display {
    padding: var(--wui-spacing-s) var(--wui-spacing-m);
    border-radius: var(--wui-border-radius-s);
    background-color: var(--wui-color-bg-125);
    margin-top: var(--wui-spacing-s);
    margin-bottom: var(--wui-spacing-s);
  }

  .token-display wui-text {
    text-transform: none;
  }

  wui-loading-spinner {
    padding: var(--wui-spacing-xs);
  }
`;var J=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let X=class extends i.WF{constructor(){super(),this.unsubscribe=[],this.amount="",this.tokenSymbol="",this.networkName="",this.exchanges=G.state.exchanges,this.isLoading=G.state.isLoading,this.loadingExchangeId=null,this.connectedWalletInfo=n.U.state.connectedWalletInfo,this.initializePaymentDetails(),this.unsubscribe.push(G.subscribeKey("exchanges",e=>this.exchanges=e)),this.unsubscribe.push(G.subscribeKey("isLoading",e=>this.isLoading=e)),this.unsubscribe.push(n.U.subscribe(e=>this.connectedWalletInfo=e.connectedWalletInfo)),G.fetchExchanges()}get isWalletConnected(){return"connected"===n.U.state.status}render(){return(0,i.qy)`
      <wui-flex flexDirection="column">
        <wui-flex flexDirection="column" .padding=${["0","l","l","l"]} gap="s">
          ${this.renderPaymentHeader()}

          <wui-flex flexDirection="column" gap="s">
            ${this.renderPayWithWallet()} ${this.renderExchangeOptions()}
          </wui-flex>
        </wui-flex>
      </wui-flex>
    `}initializePaymentDetails(){let e=G.getPaymentAsset();this.networkName=e.network,this.tokenSymbol=e.metadata.symbol,this.amount=G.state.amount.toString()}renderPayWithWallet(){return!function(e){let{chainNamespace:t}=y.C.parseCaipNetworkId(e);return j.includes(t)}(this.networkName)?(0,i.qy)``:(0,i.qy)`<wui-flex flexDirection="column" gap="s">
        ${this.isWalletConnected?this.renderConnectedView():this.renderDisconnectedView()}
      </wui-flex>
      <wui-separator text="or"></wui-separator>`}renderPaymentHeader(){let e=this.networkName;if(this.networkName){let t=s.W.getAllRequestedCaipNetworks().find(e=>e.caipNetworkId===this.networkName);t&&(e=t.name)}return(0,i.qy)`
      <wui-flex flexDirection="column" alignItems="center">
        <wui-flex alignItems="center" gap="xs">
          <wui-text variant="large-700" color="fg-100">${this.amount||"0.0000"}</wui-text>
          <wui-flex class="token-display" alignItems="center" gap="xxs">
            <wui-text variant="paragraph-600" color="fg-100">
              ${this.tokenSymbol||"Unknown Asset"}
            </wui-text>
            ${e?(0,i.qy)`
                  <wui-text variant="small-500" color="fg-200"> on ${e} </wui-text>
                `:""}
          </wui-flex>
        </wui-flex>
      </wui-flex>
    `}renderConnectedView(){let e=this.connectedWalletInfo?.name||"connected wallet";return(0,i.qy)`
      <wui-list-item
        @click=${this.onWalletPayment}
        ?chevron=${!0}
        data-testid="wallet-payment-option"
      >
        <wui-flex alignItems="center" gap="s">
          <wui-wallet-image
            size="sm"
            imageSrc=${(0,o.J)(this.connectedWalletInfo?.icon)}
            name=${(0,o.J)(this.connectedWalletInfo?.name)}
          ></wui-wallet-image>
          <wui-text variant="paragraph-500" color="inherit">Pay with ${e}</wui-text>
        </wui-flex>
      </wui-list-item>

      <wui-list-item
        variant="icon"
        iconVariant="overlay"
        icon="disconnect"
        @click=${this.onDisconnect}
        data-testid="disconnect-button"
        ?chevron=${!1}
      >
        <wui-text variant="paragraph-500" color="fg-200">Disconnect</wui-text>
      </wui-list-item>
    `}renderDisconnectedView(){return(0,i.qy)`<wui-list-item
      variant="icon"
      iconVariant="overlay"
      icon="walletPlaceholder"
      @click=${this.onWalletPayment}
      ?chevron=${!0}
      data-testid="wallet-payment-option"
    >
      <wui-text variant="paragraph-500" color="inherit">Pay from wallet</wui-text>
    </wui-list-item>`}renderExchangeOptions(){return this.isLoading?(0,i.qy)`<wui-flex justifyContent="center" alignItems="center">
        <wui-spinner size="md"></wui-spinner>
      </wui-flex>`:0===this.exchanges.length?(0,i.qy)`<wui-flex justifyContent="center" alignItems="center">
        <wui-text variant="paragraph-500" color="fg-100">No exchanges available</wui-text>
      </wui-flex>`:this.exchanges.map(e=>(0,i.qy)`
        <wui-list-item
          @click=${()=>this.onExchangePayment(e.id)}
          data-testid="exchange-option-${e.id}"
          ?chevron=${!0}
          ?disabled=${null!==this.loadingExchangeId}
        >
          <wui-flex alignItems="center" gap="s">
            ${this.loadingExchangeId===e.id?(0,i.qy)`<wui-loading-spinner color="accent-100" size="md"></wui-loading-spinner>`:(0,i.qy)`<wui-wallet-image
                  size="sm"
                  imageSrc=${(0,o.J)(e.imageUrl)}
                  name=${e.name}
                ></wui-wallet-image>`}
            <wui-text flexGrow="1" variant="paragraph-500" color="inherit"
              >Pay with ${e.name} <wui-spinner size="sm" color="fg-200"></wui-spinner
            ></wui-text>
          </wui-flex>
        </wui-list-item>
      `)}onWalletPayment(){G.handlePayWithWallet()}async onExchangePayment(e){try{this.loadingExchangeId=e;let t=await G.handlePayWithExchange(e);t&&(await c.W.open({view:"PayLoading"}),l.w.openHref(t.url,t.openInNewTab?"_blank":"_self"))}catch(e){console.error("Failed to pay with exchange",e),u.P.showError("Failed to pay with exchange")}finally{this.loadingExchangeId=null}}async onDisconnect(e){e.stopPropagation();try{await d.x.disconnect()}catch{console.error("Failed to disconnect"),u.P.showError("Failed to disconnect")}}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}};X.styles=K,J([(0,r.wk)()],X.prototype,"amount",void 0),J([(0,r.wk)()],X.prototype,"tokenSymbol",void 0),J([(0,r.wk)()],X.prototype,"networkName",void 0),J([(0,r.wk)()],X.prototype,"exchanges",void 0),J([(0,r.wk)()],X.prototype,"isLoading",void 0),J([(0,r.wk)()],X.prototype,"loadingExchangeId",void 0),J([(0,r.wk)()],X.prototype,"connectedWalletInfo",void 0),X=J([(0,p.EM)("w3m-pay-view")],X);var Q=a(69148),ee=a(11446),et=a(48789);a(15762);let ea=(0,i.AH)`
  :host {
    display: block;
    height: 100%;
    width: 100%;
  }

  wui-flex:first-child:not(:only-child) {
    position: relative;
  }

  wui-loading-thumbnail {
    position: absolute;
  }
`;var ei=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let er=class extends i.WF{constructor(){super(),this.loadingMessage="",this.subMessage="",this.paymentState="in-progress",this.paymentState=G.state.isPaymentInProgress?"in-progress":"completed",this.updateMessages(),this.setupSubscription(),this.setupExchangeSubscription()}disconnectedCallback(){clearInterval(this.exchangeSubscription)}render(){return(0,i.qy)`
      <wui-flex
        flexDirection="column"
        alignItems="center"
        .padding=${["xl","xl","xl","xl"]}
        gap="xl"
      >
        <wui-flex justifyContent="center" alignItems="center"> ${this.getStateIcon()} </wui-flex>
        <wui-flex flexDirection="column" alignItems="center" gap="xs">
          <wui-text align="center" variant="paragraph-500" color="fg-100">
            ${this.loadingMessage}
          </wui-text>
          <wui-text align="center" variant="small-400" color="fg-200">
            ${this.subMessage}
          </wui-text>
        </wui-flex>
      </wui-flex>
    `}updateMessages(){switch(this.paymentState){case"completed":this.loadingMessage="Payment completed",this.subMessage="Your transaction has been successfully processed";break;case"error":this.loadingMessage="Payment failed",this.subMessage="There was an error processing your transaction";break;default:G.state.currentPayment?.type==="exchange"?(this.loadingMessage="Payment initiated",this.subMessage="Please complete the payment on the exchange"):(this.loadingMessage="Awaiting payment confirmation",this.subMessage="Please confirm the payment transaction in your wallet")}}getStateIcon(){switch(this.paymentState){case"completed":return this.successTemplate();case"error":return this.errorTemplate();default:return this.loaderTemplate()}}setupExchangeSubscription(){G.state.currentPayment?.type==="exchange"&&(this.exchangeSubscription=setInterval(async()=>{let e=G.state.currentPayment?.exchangeId,t=G.state.currentPayment?.sessionId;e&&t&&(await G.updateBuyStatus(e,t),G.state.currentPayment?.status==="SUCCESS"&&clearInterval(this.exchangeSubscription))},4e3))}setupSubscription(){G.subscribeKey("isPaymentInProgress",e=>{e||"in-progress"!==this.paymentState||(G.state.error||!G.state.currentPayment?.result?this.paymentState="error":this.paymentState="completed",this.updateMessages(),setTimeout(()=>{"disconnected"!==d.x.state.status&&c.W.close()},3e3))}),G.subscribeKey("error",e=>{e&&"in-progress"===this.paymentState&&(this.paymentState="error",this.updateMessages())})}loaderTemplate(){let e=Q.W.state.themeVariables["--w3m-border-radius-master"],t=e?parseInt(e.replace("px",""),10):4,a=this.getPaymentIcon();return(0,i.qy)`
      <wui-flex justifyContent="center" alignItems="center" style="position: relative;">
        ${a?(0,i.qy)`<wui-wallet-image size="lg" imageSrc=${a}></wui-wallet-image>`:null}
        <wui-loading-thumbnail radius=${9*t}></wui-loading-thumbnail>
      </wui-flex>
    `}getPaymentIcon(){let e=G.state.currentPayment;if(e){if("exchange"===e.type){let t=e.exchangeId;if(t){let e=G.getExchangeById(t);return e?.imageUrl}}if("wallet"===e.type){let e=n.U.state.connectedWalletInfo?.icon;if(e)return e;let t=s.W.state.activeChain;if(!t)return;let a=ee.a.getConnectorId(t);if(!a)return;let i=ee.a.getConnectorById(a);if(!i)return;return et.$.getConnectorImage(i)}}}successTemplate(){return(0,i.qy)`<wui-icon size="xl" color="success-100" name="checkmark"></wui-icon>`}errorTemplate(){return(0,i.qy)`<wui-icon size="xl" color="error-100" name="close"></wui-icon>`}};async function eo(e){return G.handleOpenPay(e)}async function en(e,t=3e5){if(t<=0)throw new T(f,"Timeout must be greater than 0");try{await eo(e)}catch(e){if(e instanceof T)throw e;throw new T(E,e.message)}return new Promise((e,a)=>{var i;let r=!1,o=setTimeout(()=>{r||(r=!0,s(),a(new T(S,"Payment timeout")))},t);function n(){if(r)return;let t=G.state.currentPayment,a=G.state.error,i=G.state.isPaymentInProgress;if(t?.status==="SUCCESS"){r=!0,s(),clearTimeout(o),e({success:!0,result:t.result});return}if(t?.status==="FAILED"){r=!0,s(),clearTimeout(o),e({success:!1,error:a||"Payment failed"});return}!a||i||t||(r=!0,s(),clearTimeout(o),e({success:!1,error:a}))}let s=(i=[ed("currentPayment",n),ed("error",n),ed("isPaymentInProgress",n)],()=>{i.forEach(e=>{try{e()}catch{}})});n()})}function es(){return G.getExchanges()}function ec(){return G.state.currentPayment?.result}function el(){return G.state.error}function eu(){return G.state.isPaymentInProgress}function ed(e,t){return G.subscribeKey(e,t)}er.styles=ea,ei([(0,r.wk)()],er.prototype,"loadingMessage",void 0),ei([(0,r.wk)()],er.prototype,"subMessage",void 0),ei([(0,r.wk)()],er.prototype,"paymentState",void 0),er=ei([(0,p.EM)("w3m-pay-loading-view")],er);let ep={network:"eip155:8453",asset:"native",metadata:{name:"Ethereum",symbol:"ETH",decimals:18}},eh={network:"eip155:8453",asset:"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",metadata:{name:"USD Coin",symbol:"USDC",decimals:6}},eg={network:"eip155:84532",asset:"native",metadata:{name:"Ethereum",symbol:"ETH",decimals:18}},ew={network:"eip155:1",asset:"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",metadata:{name:"USD Coin",symbol:"USDC",decimals:6}},ey={network:"eip155:10",asset:"0x0b2c639c533813f4aa9d7837caf62653d097ff85",metadata:{name:"USD Coin",symbol:"USDC",decimals:6}},ev={network:"eip155:42161",asset:"0xaf88d065e77c8cC2239327C5EDb3A432268e5831",metadata:{name:"USD Coin",symbol:"USDC",decimals:6}},em={network:"eip155:137",asset:"0x3c499c542cef5e3811e1192ce70d8cc03d5c3359",metadata:{name:"USD Coin",symbol:"USDC",decimals:6}},eb={network:"solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp",asset:"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",metadata:{name:"USD Coin",symbol:"USDC",decimals:6}},ef={network:"eip155:1",asset:"0xdAC17F958D2ee523a2206206994597C13D831ec7",metadata:{name:"Tether USD",symbol:"USDT",decimals:6}},ex={network:"eip155:10",asset:"0x94b008aA00579c1307B0EF2c499aD98a8ce58e58",metadata:{name:"Tether USD",symbol:"USDT",decimals:6}},ek={network:"eip155:42161",asset:"0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9",metadata:{name:"Tether USD",symbol:"USDT",decimals:6}},eP={network:"eip155:137",asset:"0xc2132d05d31c914a87c6611c10748aeb04b58e8f",metadata:{name:"Tether USD",symbol:"USDT",decimals:6}},eE={network:"solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp",asset:"Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB",metadata:{name:"Tether USD",symbol:"USDT",decimals:6}},eI={network:"solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp",asset:"native",metadata:{name:"Solana",symbol:"SOL",decimals:9}}},61327:(e,t,a)=>{var i=a(88088),r=a(53227),o=a(40859);a(19763),a(88364),a(84915),a(12936),a(24621);var n=a(94895),s=a(69428);a(15642);let c=(0,i.AH)`
  button {
    column-gap: var(--wui-spacing-s);
    padding: 11px 18px 11px var(--wui-spacing-s);
    width: 100%;
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xs);
    color: var(--wui-color-fg-250);
    transition:
      color var(--wui-ease-out-power-1) var(--wui-duration-md),
      background-color var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: color, background-color;
  }

  button[data-iconvariant='square'],
  button[data-iconvariant='square-blue'] {
    padding: 6px 18px 6px 9px;
  }

  button > wui-flex {
    flex: 1;
  }

  button > wui-image {
    width: 32px;
    height: 32px;
    box-shadow: 0 0 0 2px var(--wui-color-gray-glass-005);
    border-radius: var(--wui-border-radius-3xl);
  }

  button > wui-icon {
    width: 36px;
    height: 36px;
    transition: opacity var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: opacity;
  }

  button > wui-icon-box[data-variant='blue'] {
    box-shadow: 0 0 0 2px var(--wui-color-accent-glass-005);
  }

  button > wui-icon-box[data-variant='overlay'] {
    box-shadow: 0 0 0 2px var(--wui-color-gray-glass-005);
  }

  button > wui-icon-box[data-variant='square-blue'] {
    border-radius: var(--wui-border-radius-3xs);
    position: relative;
    border: none;
    width: 36px;
    height: 36px;
  }

  button > wui-icon-box[data-variant='square-blue']::after {
    content: '';
    position: absolute;
    top: 0;
    bottom: 0;
    left: 0;
    right: 0;
    border-radius: inherit;
    border: 1px solid var(--wui-color-accent-glass-010);
    pointer-events: none;
  }

  button > wui-icon:last-child {
    width: 14px;
    height: 14px;
  }

  button:disabled {
    color: var(--wui-color-gray-glass-020);
  }

  button[data-loading='true'] > wui-icon {
    opacity: 0;
  }

  wui-loading-spinner {
    position: absolute;
    right: 18px;
    top: 50%;
    transform: translateY(-50%);
  }
`;var l=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let u=class extends i.WF{constructor(){super(...arguments),this.tabIdx=void 0,this.variant="icon",this.disabled=!1,this.imageSrc=void 0,this.alt=void 0,this.chevron=!1,this.loading=!1}render(){return(0,i.qy)`
      <button
        ?disabled=${!!this.loading||!!this.disabled}
        data-loading=${this.loading}
        data-iconvariant=${(0,o.J)(this.iconVariant)}
        tabindex=${(0,o.J)(this.tabIdx)}
      >
        ${this.loadingTemplate()} ${this.visualTemplate()}
        <wui-flex gap="3xs">
          <slot></slot>
        </wui-flex>
        ${this.chevronTemplate()}
      </button>
    `}visualTemplate(){if("image"===this.variant&&this.imageSrc)return(0,i.qy)`<wui-image src=${this.imageSrc} alt=${this.alt??"list item"}></wui-image>`;if("square"===this.iconVariant&&this.icon&&"icon"===this.variant)return(0,i.qy)`<wui-icon name=${this.icon}></wui-icon>`;if("icon"===this.variant&&this.icon&&this.iconVariant){let e=["blue","square-blue"].includes(this.iconVariant)?"accent-100":"fg-200",t="square-blue"===this.iconVariant?"mdl":"md",a=this.iconSize?this.iconSize:t;return(0,i.qy)`
        <wui-icon-box
          data-variant=${this.iconVariant}
          icon=${this.icon}
          iconSize=${a}
          background="transparent"
          iconColor=${e}
          backgroundColor=${e}
          size=${t}
        ></wui-icon-box>
      `}return null}loadingTemplate(){return this.loading?(0,i.qy)`<wui-loading-spinner
        data-testid="wui-list-item-loading-spinner"
        color="fg-300"
      ></wui-loading-spinner>`:(0,i.qy)``}chevronTemplate(){return this.chevron?(0,i.qy)`<wui-icon size="inherit" color="fg-200" name="chevronRight"></wui-icon>`:null}};u.styles=[n.W5,n.fD,c],l([(0,r.MZ)()],u.prototype,"icon",void 0),l([(0,r.MZ)()],u.prototype,"iconSize",void 0),l([(0,r.MZ)()],u.prototype,"tabIdx",void 0),l([(0,r.MZ)()],u.prototype,"variant",void 0),l([(0,r.MZ)()],u.prototype,"iconVariant",void 0),l([(0,r.MZ)({type:Boolean})],u.prototype,"disabled",void 0),l([(0,r.MZ)()],u.prototype,"imageSrc",void 0),l([(0,r.MZ)()],u.prototype,"alt",void 0),l([(0,r.MZ)({type:Boolean})],u.prototype,"chevron",void 0),l([(0,r.MZ)({type:Boolean})],u.prototype,"loading",void 0),u=l([(0,s.E)("wui-list-item")],u)},66449:(e,t,a)=>{a(82989)},80124:(e,t,a)=>{var i=a(88088),r=a(53227);let o=(0,i.JW)`<svg width="86" height="96" fill="none">
  <path
    d="M78.3244 18.926L50.1808 2.45078C45.7376 -0.150261 40.2624 -0.150262 35.8192 2.45078L7.6756 18.926C3.23322 21.5266 0.5 26.3301 0.5 31.5248V64.4752C0.5 69.6699 3.23322 74.4734 7.6756 77.074L35.8192 93.5492C40.2624 96.1503 45.7376 96.1503 50.1808 93.5492L78.3244 77.074C82.7668 74.4734 85.5 69.6699 85.5 64.4752V31.5248C85.5 26.3301 82.7668 21.5266 78.3244 18.926Z"
  />
</svg>`;var n=a(31320);let s=(0,i.JW)`
  <svg fill="none" viewBox="0 0 36 40">
    <path
      d="M15.4 2.1a5.21 5.21 0 0 1 5.2 0l11.61 6.7a5.21 5.21 0 0 1 2.61 4.52v13.4c0 1.87-1 3.59-2.6 4.52l-11.61 6.7c-1.62.93-3.6.93-5.22 0l-11.6-6.7a5.21 5.21 0 0 1-2.61-4.51v-13.4c0-1.87 1-3.6 2.6-4.52L15.4 2.1Z"
    />
  </svg>
`;a(19763),a(88364);var c=a(94895),l=a(69428);let u=(0,i.AH)`
  :host {
    position: relative;
    border-radius: inherit;
    display: flex;
    justify-content: center;
    align-items: center;
    width: var(--local-width);
    height: var(--local-height);
  }

  :host([data-round='true']) {
    background: var(--wui-color-gray-glass-002);
    border-radius: 100%;
    outline: 1px solid var(--wui-color-gray-glass-005);
  }

  svg {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    z-index: 1;
    fill: var(--wui-color-gray-glass-002);
  }

  svg > path {
    stroke: var(--local-stroke);
  }

  wui-image {
    width: 100%;
    height: 100%;
    -webkit-clip-path: var(--local-path);
    clip-path: var(--local-path);
    background: var(--wui-color-gray-glass-002);
  }

  wui-icon {
    transform: translateY(-5%);
    width: var(--local-icon-size);
    height: var(--local-icon-size);
  }
`;var d=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let p=class extends i.WF{constructor(){super(...arguments),this.size="md",this.name="uknown",this.networkImagesBySize={sm:s,md:n.a,lg:o},this.selected=!1,this.round=!1}render(){return this.round?(this.dataset.round="true",this.style.cssText=`
      --local-width: var(--wui-spacing-3xl);
      --local-height: var(--wui-spacing-3xl);
      --local-icon-size: var(--wui-spacing-l);
    `):this.style.cssText=`

      --local-path: var(--wui-path-network-${this.size});
      --local-width:  var(--wui-width-network-${this.size});
      --local-height:  var(--wui-height-network-${this.size});
      --local-icon-size:  var(--wui-icon-size-network-${this.size});
    `,(0,i.qy)`${this.templateVisual()} ${this.svgTemplate()} `}svgTemplate(){return this.round?null:this.networkImagesBySize[this.size]}templateVisual(){return this.imageSrc?(0,i.qy)`<wui-image src=${this.imageSrc} alt=${this.name}></wui-image>`:(0,i.qy)`<wui-icon size="inherit" color="fg-200" name="networkPlaceholder"></wui-icon>`}};p.styles=[c.W5,u],d([(0,r.MZ)()],p.prototype,"size",void 0),d([(0,r.MZ)()],p.prototype,"name",void 0),d([(0,r.MZ)({type:Object})],p.prototype,"networkImagesBySize",void 0),d([(0,r.MZ)()],p.prototype,"imageSrc",void 0),d([(0,r.MZ)({type:Boolean})],p.prototype,"selected",void 0),d([(0,r.MZ)({type:Boolean})],p.prototype,"round",void 0),p=d([(0,l.E)("wui-network-image")],p)},82989:(e,t,a)=>{var i=a(88088),r=a(53227);a(84915),a(12936);var o=a(94895),n=a(69428);let s=(0,i.AH)`
  :host {
    width: var(--local-width);
    position: relative;
  }

  button {
    border: none;
    border-radius: var(--local-border-radius);
    width: var(--local-width);
    white-space: nowrap;
  }

  /* -- Sizes --------------------------------------------------- */
  button[data-size='xs'] {
    padding: var(--wui-spacing-3xs) var(--wui-spacing-s) var(--wui-spacing-3xs) var(--wui-spacing-s);
    height: 24px;
  }

  button[data-size='xs'][data-icon-left='true'][data-icon-right='false'] {
    padding: var(--wui-spacing-3xs) var(--wui-spacing-s) var(--wui-spacing-3xs) var(--wui-spacing-s);
  }

  button[data-size='xs'][data-icon-right='true'][data-icon-left='false'] {
    padding: var(--wui-spacing-3xs) var(--wui-spacing-s) var(--wui-spacing-3xs) var(--wui-spacing-s);
  }

  button[data-size='sm'] {
    padding: 7.2px var(--wui-spacing-s) 7.2px var(--wui-spacing-s);
    height: 32px;
  }

  button[data-size='sm'][data-icon-left='true'][data-icon-right='false'] {
    padding: 7.2px var(--wui-spacing-s) 7.2px var(--wui-spacing-s);
  }

  button[data-size='sm'][data-icon-right='true'][data-icon-left='false'] {
    padding: 7.2px var(--wui-spacing-s) 7.2px var(--wui-spacing-s);
  }

  button[data-size='md'] {
    padding: 8.2px var(--wui-spacing-l) 9px var(--wui-spacing-l);
    height: 36px;
  }

  button[data-size='md'][data-icon-left='true'][data-icon-right='false'] {
    padding: 8.2px var(--wui-spacing-l) 9px var(--wui-spacing-s);
  }

  button[data-size='md'][data-icon-right='true'][data-icon-left='false'] {
    padding: 8.2px var(--wui-spacing-s) 9px var(--wui-spacing-l);
  }

  button[data-size='lg'] {
    padding: var(--wui-spacing-m) var(--wui-spacing-2l);
    height: 48px;
  }

  /* -- Variants --------------------------------------------------------- */
  button[data-variant='main'] {
    background-color: var(--wui-color-accent-100);
    color: var(--wui-color-inverse-100);
    border: none;
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-010);
  }

  button[data-variant='inverse'] {
    background-color: var(--wui-color-inverse-100);
    color: var(--wui-color-inverse-000);
    border: none;
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-010);
  }

  button[data-variant='accent'] {
    background-color: var(--wui-color-accent-glass-010);
    color: var(--wui-color-accent-100);
    border: none;
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-005);
  }

  button[data-variant='accent-error'] {
    background: var(--wui-color-error-glass-015);
    color: var(--wui-color-error-100);
    border: none;
    box-shadow: inset 0 0 0 1px var(--wui-color-error-glass-010);
  }

  button[data-variant='accent-success'] {
    background: var(--wui-color-success-glass-015);
    color: var(--wui-color-success-100);
    border: none;
    box-shadow: inset 0 0 0 1px var(--wui-color-success-glass-010);
  }

  button[data-variant='neutral'] {
    background: transparent;
    color: var(--wui-color-fg-100);
    border: none;
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-005);
  }

  button[data-variant='shade'] {
    background: var(--wui-color-gray-glass-002);
    color: var(--wui-color-fg-200);
    border: none;
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-005);
  }

  /* -- Focus states --------------------------------------------------- */
  button[data-variant='main']:focus-visible:enabled {
    background-color: var(--wui-color-accent-090);
    box-shadow:
      inset 0 0 0 1px var(--wui-color-accent-100),
      0 0 0 4px var(--wui-color-accent-glass-020);
  }
  button[data-variant='inverse']:focus-visible:enabled {
    background-color: var(--wui-color-inverse-100);
    box-shadow:
      inset 0 0 0 1px var(--wui-color-gray-glass-010),
      0 0 0 4px var(--wui-color-accent-glass-020);
  }
  button[data-variant='accent']:focus-visible:enabled {
    background-color: var(--wui-color-accent-glass-010);
    box-shadow:
      inset 0 0 0 1px var(--wui-color-accent-100),
      0 0 0 4px var(--wui-color-accent-glass-020);
  }
  button[data-variant='accent-error']:focus-visible:enabled {
    background: var(--wui-color-error-glass-015);
    box-shadow:
      inset 0 0 0 1px var(--wui-color-error-100),
      0 0 0 4px var(--wui-color-error-glass-020);
  }
  button[data-variant='accent-success']:focus-visible:enabled {
    background: var(--wui-color-success-glass-015);
    box-shadow:
      inset 0 0 0 1px var(--wui-color-success-100),
      0 0 0 4px var(--wui-color-success-glass-020);
  }
  button[data-variant='neutral']:focus-visible:enabled {
    background: var(--wui-color-gray-glass-005);
    box-shadow:
      inset 0 0 0 1px var(--wui-color-gray-glass-010),
      0 0 0 4px var(--wui-color-gray-glass-002);
  }
  button[data-variant='shade']:focus-visible:enabled {
    background: var(--wui-color-gray-glass-005);
    box-shadow:
      inset 0 0 0 1px var(--wui-color-gray-glass-010),
      0 0 0 4px var(--wui-color-gray-glass-002);
  }

  /* -- Hover & Active states ----------------------------------------------------------- */
  @media (hover: hover) and (pointer: fine) {
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

    button[data-variant='accent-error']:hover:enabled {
      background: var(--wui-color-error-glass-020);
      color: var(--wui-color-error-100);
    }

    button[data-variant='accent-error']:active:enabled {
      background: var(--wui-color-error-glass-030);
      color: var(--wui-color-error-100);
    }

    button[data-variant='accent-success']:hover:enabled {
      background: var(--wui-color-success-glass-020);
      color: var(--wui-color-success-100);
    }

    button[data-variant='accent-success']:active:enabled {
      background: var(--wui-color-success-glass-030);
      color: var(--wui-color-success-100);
    }

    button[data-variant='neutral']:hover:enabled {
      background: var(--wui-color-gray-glass-002);
    }

    button[data-variant='neutral']:active:enabled {
      background: var(--wui-color-gray-glass-005);
    }

    button[data-size='lg'][data-icon-left='true'][data-icon-right='false'] {
      padding-left: var(--wui-spacing-m);
    }

    button[data-size='lg'][data-icon-right='true'][data-icon-left='false'] {
      padding-right: var(--wui-spacing-m);
    }

    button[data-variant='shade']:hover:enabled {
      background: var(--wui-color-gray-glass-002);
    }

    button[data-variant='shade']:active:enabled {
      background: var(--wui-color-gray-glass-005);
    }
  }

  /* -- Disabled state --------------------------------------------------- */
  button:disabled {
    background-color: var(--wui-color-gray-glass-002);
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-002);
    color: var(--wui-color-gray-glass-020);
    cursor: not-allowed;
  }

  button > wui-text {
    transition: opacity var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: opacity;
    opacity: var(--local-opacity-100);
  }

  ::slotted(*) {
    transition: opacity var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: opacity;
    opacity: var(--local-opacity-100);
  }

  wui-loading-spinner {
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    opacity: var(--local-opacity-000);
  }
`;var c=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let l={main:"inverse-100",inverse:"inverse-000",accent:"accent-100","accent-error":"error-100","accent-success":"success-100",neutral:"fg-100",disabled:"gray-glass-020",shade:"fg-100"},u={lg:"paragraph-600",md:"small-600",sm:"small-600",xs:"tiny-600"},d={lg:"md",md:"md",sm:"sm",xs:"sm"},p=class extends i.WF{constructor(){super(...arguments),this.size="lg",this.disabled=!1,this.fullWidth=!1,this.loading=!1,this.variant="main",this.hasIconLeft=!1,this.hasIconRight=!1,this.borderRadius="m"}render(){this.style.cssText=`
    --local-width: ${this.fullWidth?"100%":"auto"};
    --local-opacity-100: ${+!this.loading};
    --local-opacity-000: ${+!!this.loading};
    --local-border-radius: var(--wui-border-radius-${this.borderRadius});
    `;let e=this.textVariant??u[this.size];return(0,i.qy)`
      <button
        data-variant=${this.variant}
        data-icon-left=${this.hasIconLeft}
        data-icon-right=${this.hasIconRight}
        data-size=${this.size}
        ?disabled=${this.disabled}
      >
        ${this.loadingTemplate()}
        <slot name="iconLeft" @slotchange=${()=>this.handleSlotLeftChange()}></slot>
        <wui-text variant=${e} color="inherit">
          <slot></slot>
        </wui-text>
        <slot name="iconRight" @slotchange=${()=>this.handleSlotRightChange()}></slot>
      </button>
    `}handleSlotLeftChange(){this.hasIconLeft=!0}handleSlotRightChange(){this.hasIconRight=!0}loadingTemplate(){if(this.loading){let e=d[this.size],t=this.disabled?l.disabled:l[this.variant];return(0,i.qy)`<wui-loading-spinner color=${t} size=${e}></wui-loading-spinner>`}return(0,i.qy)``}};p.styles=[o.W5,o.fD,s],c([(0,r.MZ)()],p.prototype,"size",void 0),c([(0,r.MZ)({type:Boolean})],p.prototype,"disabled",void 0),c([(0,r.MZ)({type:Boolean})],p.prototype,"fullWidth",void 0),c([(0,r.MZ)({type:Boolean})],p.prototype,"loading",void 0),c([(0,r.MZ)()],p.prototype,"variant",void 0),c([(0,r.MZ)({type:Boolean})],p.prototype,"hasIconLeft",void 0),c([(0,r.MZ)({type:Boolean})],p.prototype,"hasIconRight",void 0),c([(0,r.MZ)()],p.prototype,"borderRadius",void 0),c([(0,r.MZ)()],p.prototype,"textVariant",void 0),p=c([(0,n.E)("wui-button")],p)},84915:(e,t,a)=>{var i=a(88088),r=a(53227),o=a(94895),n=a(69428);let s=(0,i.AH)`
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
`;var c=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let l=class extends i.WF{constructor(){super(...arguments),this.color="accent-100",this.size="lg"}render(){return this.style.cssText=`--local-color: ${"inherit"===this.color?"inherit":`var(--wui-color-${this.color})`}`,this.dataset.size=this.size,(0,i.qy)`<svg viewBox="25 25 50 50">
      <circle r="20" cy="50" cx="50"></circle>
    </svg>`}};l.styles=[o.W5,s],c([(0,r.MZ)()],l.prototype,"color",void 0),c([(0,r.MZ)()],l.prototype,"size",void 0),l=c([(0,n.E)("wui-loading-spinner")],l)},88364:(e,t,a)=>{var i=a(88088),r=a(53227),o=a(94895),n=a(69428);let s=(0,i.AH)`
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
`;var c=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let l=class extends i.WF{constructor(){super(...arguments),this.src="./path/to/image.jpg",this.alt="Image",this.size=void 0,this.objectFit="cover"}render(){return this.objectFit&&(this.dataset.objectFit=this.objectFit),this.style.cssText=`
      --local-width: ${this.size?`var(--wui-icon-size-${this.size});`:"100%"};
      --local-height: ${this.size?`var(--wui-icon-size-${this.size});`:"100%"};
      `,(0,i.qy)`<img src=${this.src} alt=${this.alt} @error=${this.handleImageError} />`}handleImageError(){this.dispatchEvent(new CustomEvent("onLoadError",{bubbles:!0,composed:!0}))}};l.styles=[o.W5,o.ck,s],c([(0,r.MZ)()],l.prototype,"src",void 0),c([(0,r.MZ)()],l.prototype,"alt",void 0),c([(0,r.MZ)()],l.prototype,"size",void 0),c([(0,r.MZ)()],l.prototype,"objectFit",void 0),l=c([(0,n.E)("wui-image")],l)},88844:(e,t,a)=>{a(84915)},92730:(e,t,a)=>{var i=a(88088),r=a(53227);a(19763);var o=a(94895),n=a(69428);let s=(0,i.AH)`
  :host {
    position: relative;
  }

  button {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 48px;
    width: 100%;
    background-color: var(--wui-color-accent-glass-010);
    border-radius: var(--wui-border-radius-xs);
    border: 1px solid var(--wui-color-accent-glass-010);
    transition: background-color var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: background-color;
  }

  wui-tooltip {
    padding: 7px var(--wui-spacing-s) 8px var(--wui-spacing-s);
    position: absolute;
    top: -8px;
    left: 50%;
    transform: translate(-50%, -100%);
    opacity: 0;
    display: none;
  }

  @media (hover: hover) and (pointer: fine) {
    button:hover:enabled {
      background-color: var(--wui-color-accent-glass-015);
    }

    button:active:enabled {
      background-color: var(--wui-color-accent-glass-020);
    }
  }
`;var c=function(e,t,a,i){var r,o=arguments.length,n=o<3?t:null===i?i=Object.getOwnPropertyDescriptor(t,a):i;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,a,i);else for(var s=e.length-1;s>=0;s--)(r=e[s])&&(n=(o<3?r(n):o>3?r(t,a,n):r(t,a))||n);return o>3&&n&&Object.defineProperty(t,a,n),n};let l=class extends i.WF{constructor(){super(...arguments),this.text="",this.icon="card"}render(){return(0,i.qy)`<button>
      <wui-icon color="accent-100" name=${this.icon} size="lg"></wui-icon>
    </button>`}};l.styles=[o.W5,o.fD,s],c([(0,r.MZ)()],l.prototype,"text",void 0),c([(0,r.MZ)()],l.prototype,"icon",void 0),l=c([(0,n.E)("wui-icon-button")],l)},94548:(e,t,a)=>{a(19763)}}]);