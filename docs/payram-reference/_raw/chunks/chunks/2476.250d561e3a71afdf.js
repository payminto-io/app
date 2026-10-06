"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[2476],{15642:(t,e,a)=>{var o=a(88088),r=a(53227);a(19763);var i=a(94895),s=a(69428);let n=(0,o.AH)`
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
`;var c=function(t,e,a,o){var r,i=arguments.length,s=i<3?e:null===o?o=Object.getOwnPropertyDescriptor(e,a):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,a,o);else for(var n=t.length-1;n>=0;n--)(r=t[n])&&(s=(i<3?r(s):i>3?r(e,a,s):r(e,a))||s);return i>3&&s&&Object.defineProperty(e,a,s),s};let l=class extends o.WF{constructor(){super(...arguments),this.size="md",this.backgroundColor="accent-100",this.iconColor="accent-100",this.background="transparent",this.border=!1,this.borderColor="wui-color-bg-125",this.icon="copy"}render(){let t=this.iconSize||this.size,e="lg"===this.size,a="xl"===this.size,r="gray"===this.background,i="opaque"===this.background,s="accent-100"===this.backgroundColor&&i||"success-100"===this.backgroundColor&&i||"error-100"===this.backgroundColor&&i||"inverse-100"===this.backgroundColor&&i,n=`var(--wui-color-${this.backgroundColor})`;return s?n=`var(--wui-icon-box-bg-${this.backgroundColor})`:r&&(n=`var(--wui-color-gray-${this.backgroundColor})`),this.style.cssText=`
       --local-bg-value: ${n};
       --local-bg-mix: ${s||r?"100%":e?"12%":"16%"};
       --local-border-radius: var(--wui-border-radius-${e?"xxs":a?"s":"3xl"});
       --local-size: var(--wui-icon-box-size-${this.size});
       --local-border: ${"wui-color-bg-125"===this.borderColor?"2px":"1px"} solid ${this.border?`var(--${this.borderColor})`:"transparent"}
   `,(0,o.qy)` <wui-icon color=${this.iconColor} size=${t} name=${this.icon}></wui-icon> `}};l.styles=[i.W5,i.fD,n],c([(0,r.MZ)()],l.prototype,"size",void 0),c([(0,r.MZ)()],l.prototype,"backgroundColor",void 0),c([(0,r.MZ)()],l.prototype,"iconColor",void 0),c([(0,r.MZ)()],l.prototype,"iconSize",void 0),c([(0,r.MZ)()],l.prototype,"background",void 0),c([(0,r.MZ)({type:Boolean})],l.prototype,"border",void 0),c([(0,r.MZ)()],l.prototype,"borderColor",void 0),c([(0,r.MZ)()],l.prototype,"icon",void 0),l=c([(0,s.E)("wui-icon-box")],l)},20903:(t,e,a)=>{var o=a(88088),r=a(53227);a(19763);var i=a(94895),s=a(69428);let n=(0,o.AH)`
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
`;var c=function(t,e,a,o){var r,i=arguments.length,s=i<3?e:null===o?o=Object.getOwnPropertyDescriptor(e,a):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,a,o);else for(var n=t.length-1;n>=0;n--)(r=t[n])&&(s=(i<3?r(s):i>3?r(e,a,s):r(e,a))||s);return i>3&&s&&Object.defineProperty(e,a,s),s};let l=class extends o.WF{constructor(){super(...arguments),this.size="md",this.disabled=!1,this.icon="copy",this.iconColor="inherit"}render(){this.dataset.size=this.size;let t="",e="";switch(this.size){case"lg":t="--wui-border-radius-xs",e="--wui-spacing-1xs";break;case"sm":t="--wui-border-radius-3xs",e="--wui-spacing-xxs";break;default:t="--wui-border-radius-xxs",e="--wui-spacing-2xs"}return this.style.cssText=`
    --local-border-radius: var(${t});
    --local-padding: var(${e});
    `,(0,o.qy)`
      <button ?disabled=${this.disabled}>
        <wui-icon color=${this.iconColor} size=${this.size} name=${this.icon}></wui-icon>
      </button>
    `}};l.styles=[i.W5,i.fD,i.ck,n],c([(0,r.MZ)()],l.prototype,"size",void 0),c([(0,r.MZ)({type:Boolean})],l.prototype,"disabled",void 0),c([(0,r.MZ)()],l.prototype,"icon",void 0),c([(0,r.MZ)()],l.prototype,"iconColor",void 0),l=c([(0,s.E)("wui-icon-link")],l)},26400:(t,e,a)=>{a(63270)},35787:(t,e,a)=>{a(20903)},48342:(t,e,a)=>{a(88364)},61327:(t,e,a)=>{var o=a(88088),r=a(53227),i=a(40859);a(19763),a(88364),a(84915),a(12936),a(24621);var s=a(94895),n=a(69428);a(15642);let c=(0,o.AH)`
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
`;var l=function(t,e,a,o){var r,i=arguments.length,s=i<3?e:null===o?o=Object.getOwnPropertyDescriptor(e,a):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,a,o);else for(var n=t.length-1;n>=0;n--)(r=t[n])&&(s=(i<3?r(s):i>3?r(e,a,s):r(e,a))||s);return i>3&&s&&Object.defineProperty(e,a,s),s};let u=class extends o.WF{constructor(){super(...arguments),this.tabIdx=void 0,this.variant="icon",this.disabled=!1,this.imageSrc=void 0,this.alt=void 0,this.chevron=!1,this.loading=!1}render(){return(0,o.qy)`
      <button
        ?disabled=${!!this.loading||!!this.disabled}
        data-loading=${this.loading}
        data-iconvariant=${(0,i.J)(this.iconVariant)}
        tabindex=${(0,i.J)(this.tabIdx)}
      >
        ${this.loadingTemplate()} ${this.visualTemplate()}
        <wui-flex gap="3xs">
          <slot></slot>
        </wui-flex>
        ${this.chevronTemplate()}
      </button>
    `}visualTemplate(){if("image"===this.variant&&this.imageSrc)return(0,o.qy)`<wui-image src=${this.imageSrc} alt=${this.alt??"list item"}></wui-image>`;if("square"===this.iconVariant&&this.icon&&"icon"===this.variant)return(0,o.qy)`<wui-icon name=${this.icon}></wui-icon>`;if("icon"===this.variant&&this.icon&&this.iconVariant){let t=["blue","square-blue"].includes(this.iconVariant)?"accent-100":"fg-200",e="square-blue"===this.iconVariant?"mdl":"md",a=this.iconSize?this.iconSize:e;return(0,o.qy)`
        <wui-icon-box
          data-variant=${this.iconVariant}
          icon=${this.icon}
          iconSize=${a}
          background="transparent"
          iconColor=${t}
          backgroundColor=${t}
          size=${e}
        ></wui-icon-box>
      `}return null}loadingTemplate(){return this.loading?(0,o.qy)`<wui-loading-spinner
        data-testid="wui-list-item-loading-spinner"
        color="fg-300"
      ></wui-loading-spinner>`:(0,o.qy)``}chevronTemplate(){return this.chevron?(0,o.qy)`<wui-icon size="inherit" color="fg-200" name="chevronRight"></wui-icon>`:null}};u.styles=[s.W5,s.fD,c],l([(0,r.MZ)()],u.prototype,"icon",void 0),l([(0,r.MZ)()],u.prototype,"iconSize",void 0),l([(0,r.MZ)()],u.prototype,"tabIdx",void 0),l([(0,r.MZ)()],u.prototype,"variant",void 0),l([(0,r.MZ)()],u.prototype,"iconVariant",void 0),l([(0,r.MZ)({type:Boolean})],u.prototype,"disabled",void 0),l([(0,r.MZ)()],u.prototype,"imageSrc",void 0),l([(0,r.MZ)()],u.prototype,"alt",void 0),l([(0,r.MZ)({type:Boolean})],u.prototype,"chevron",void 0),l([(0,r.MZ)({type:Boolean})],u.prototype,"loading",void 0),u=l([(0,n.E)("wui-list-item")],u)},62476:(t,e,a)=>{a.r(e),a.d(e,{W3mDepositFromExchangeView:()=>R});var o=a(88088),r=a(53227),i=a(1024),s=a(45430),n=a(3125),c=a(33452),l=a(37056),u=a(30840),d=a(19392);let p={eip155:{native:{assetNamespace:"slip44",assetReference:"60"},defaultTokenNamespace:"erc20"},solana:{native:{assetNamespace:"slip44",assetReference:"501"},defaultTokenNamespace:"token"}};class g extends Error{}async function h(t,e){let a,o=(a=d.H.getSnapshot().projectId,`https://rpc.walletconnect.org/v1/json-rpc?projectId=${a}&source=fund-wallet`),{sdkType:r,sdkVersion:i,projectId:s}=d.H.getSnapshot(),n={jsonrpc:"2.0",id:1,method:t,params:{...e||{},st:r,sv:i,projectId:s}},c=await fetch(o,{method:"POST",body:JSON.stringify(n),headers:{"Content-Type":"application/json"}}),l=await c.json();if(l.error)throw new g(l.error.message);return l}async function v(t){return(await h("reown_getExchanges",t)).result}async function b(t){return(await h("reown_getExchangePayUrl",t)).result}async function w(t){return(await h("reown_getExchangeBuyStatus",t)).result}function y(t,e){let{chainNamespace:a,chainId:o}=u.C.parseCaipNetworkId(t),r=p[a];if(!r)throw Error(`Unsupported chain namespace for CAIP-19 formatting: ${a}`);let i=r.native.assetNamespace,s=r.native.assetReference;"native"!==e&&(i=r.defaultTokenNamespace,s=e);let n=`${a}:${o}`;return`${n}/${i}:${s}`}var m=a(34066),x=a(18715),f=a(43242),k=a(44963);let P={paymentAsset:{network:"eip155:1",asset:"native",metadata:{name:"Ethereum",symbol:"ETH",decimals:0}},amount:0,tokenAmount:0,tokenPrice:null,priceLoading:!1,error:null,exchanges:[],isLoading:!1,currentPayment:void 0,isPaymentInProgress:!1,paymentId:""},$=(0,s.BX)(P),z={state:$,subscribe:t=>(0,s.B1)($,()=>t($)),subscribeKey:(t,e)=>(0,n.u$)($,t,e),resetState(){Object.assign($,{...P})},async fetchTokenPrice(){$.priceLoading=!0;let t=(0,c.K1)(),e=await x.T.fetchTokenPrice({addresses:[t]});$.tokenPrice=e.fungibles?.[0]?.price||null,$.priceLoading=!1},getTokenAmount(){if(!$.tokenPrice)throw Error("Cannot get token price");return Number(new Intl.NumberFormat("en-US",{minimumFractionDigits:0,maximumFractionDigits:4}).format($.amount/$.tokenPrice))},setAmount(t){$.amount=t,$.tokenPrice&&($.tokenAmount=this.getTokenAmount())},setPaymentAsset(t){$.paymentAsset=t},async fetchExchanges(){try{$.isLoading=!0;let t=await v({page:0,asset:y($.paymentAsset.network,$.paymentAsset.asset),amount:$.amount.toString()});$.exchanges=t.exchanges.slice(0,2)}catch(t){throw k.P.showError("Unable to get exchanges"),Error("Unable to get exchanges")}finally{$.isLoading=!1}},async getPayUrl(t,e){try{let a=Number(e.amount),o=await b({exchangeId:t,asset:y(e.network,e.asset),amount:a.toString(),recipient:`${e.network}:${e.recipient}`});return f.E.sendEvent({type:"track",event:"PAY_EXCHANGE_SELECTED",properties:{exchange:{id:t},configuration:{network:e.network,asset:e.asset,recipient:e.recipient,amount:a},currentPayment:{type:"exchange",exchangeId:t},source:"fund-from-exchange",headless:!1}}),o}catch(t){if(t instanceof Error&&t.message.includes("is not supported"))throw Error("Asset not supported");throw Error(t.message)}},async handlePayWithExchange(t){try{if(!m.U.state.address)throw Error("No account connected");$.isPaymentInProgress=!0,$.paymentId=crypto.randomUUID(),$.currentPayment={type:"exchange",exchangeId:t};let{network:e,asset:a}=$.paymentAsset,o={network:e,asset:a,amount:$.tokenAmount,recipient:m.U.state.address},r=await this.getPayUrl(t,o);if(!r)throw Error("Unable to initiate payment");$.currentPayment.sessionId=r.sessionId,$.currentPayment.status="IN_PROGRESS",$.currentPayment.exchangeId=t,l.w.openHref(r.url,"_blank","popup=yes,width=480,height=720,noopener,noreferrer")}catch(t){$.error="Unable to initiate payment",k.P.showError($.error)}},async waitUntilComplete({exchangeId:t,sessionId:e,paymentId:a,retries:o=20}){let r=await this.getBuyStatus(t,e,a);if("SUCCESS"===r.status||"FAILED"===r.status)return r;if(0===o)throw Error("Unable to get deposit status");return await new Promise(t=>{setTimeout(t,5e3)}),this.waitUntilComplete({exchangeId:t,sessionId:e,paymentId:a,retries:o-1})},async getBuyStatus(t,e,a){try{if(!$.currentPayment)throw Error("No current payment");let o=await w({sessionId:e,exchangeId:t});return $.currentPayment.status=o.status,("SUCCESS"===o.status||"FAILED"===o.status)&&($.currentPayment.result=o.txHash,$.isPaymentInProgress=!1,f.E.sendEvent({type:"track",event:"SUCCESS"===o.status?"PAY_SUCCESS":"PAY_ERROR",properties:{source:"fund-from-exchange",paymentId:a,configuration:{network:$.paymentAsset.network,asset:$.paymentAsset.asset,recipient:m.U.state.address||"",amount:$.amount},currentPayment:{type:"exchange",exchangeId:$.currentPayment?.exchangeId,sessionId:$.currentPayment?.sessionId,result:o.txHash}}})),o}catch(t){return{status:"UNKNOWN",txHash:""}}},reset(){$.currentPayment=void 0,$.isPaymentInProgress=!1,$.paymentId="",$.paymentAsset={network:"eip155:1",asset:"native",metadata:{name:"Ethereum",symbol:"ETH",decimals:0}},$.amount=0,$.tokenAmount=0,$.tokenPrice=null,$.priceLoading=!1,$.error=null,$.exchanges=[],$.isLoading=!1}};var I=a(48789),E=a(44458),C=a(59306);a(66449),a(74970),a(35787),a(48342),a(61327),a(26400),a(3658);let S=(0,o.AH)`
  .amount-input-container {
    border-radius: var(--wui-border-radius-m);
    border-top-right-radius: 0;
    border-top-left-radius: 0;
    border-bottom: 1px solid var(--wui-color-gray-glass-010);
    background-color: var(--wui-color-bg-100);
  }

  .container {
    background-color: var(--wui-color-bg-125);
  }
`;var j=function(t,e,a,o){var r,i=arguments.length,s=i<3?e:null===o?o=Object.getOwnPropertyDescriptor(e,a):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,a,o);else for(var n=t.length-1;n>=0;n--)(r=t[n])&&(s=(i<3?r(s):i>3?r(e,a,s):r(e,a))||s);return i>3&&s&&Object.defineProperty(e,a,s),s};let A=[10,50,100],R=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.network=i.W.state.activeCaipNetwork,this.exchanges=z.state.exchanges,this.isLoading=z.state.isLoading,this.amount=z.state.amount,this.tokenAmount=z.state.tokenAmount,this.priceLoading=z.state.priceLoading,this.isPaymentInProgress=z.state.isPaymentInProgress,this.currentPayment=z.state.currentPayment,this.paymentId=z.state.paymentId,this.unsubscribe.push(z.subscribe(t=>{this.exchanges=t.exchanges,this.isLoading=t.isLoading,this.amount=t.amount,this.tokenAmount=t.tokenAmount,this.priceLoading=t.priceLoading,this.paymentId=t.paymentId,this.isPaymentInProgress=t.isPaymentInProgress,this.currentPayment=t.currentPayment,t.isPaymentInProgress&&t.currentPayment?.exchangeId&&t.currentPayment?.sessionId&&t.paymentId&&this.handlePaymentInProgress()}))}disconnectedCallback(){this.unsubscribe.forEach(t=>t()),z.reset()}firstUpdated(){z.fetchExchanges(),z.fetchTokenPrice()}render(){return(0,o.qy)`
      <wui-flex flexDirection="column" gap="xs" class="container">
        ${this.amountInputTemplate()} ${this.exchangesTemplate()}
      </wui-flex>
    `}exchangesTemplate(){return(0,o.qy)`
      <wui-flex
        flexDirection="column"
        gap="xs"
        .padding=${["xs","s","s","s"]}
        class="exchanges-container"
      >
        ${this.exchanges.map(t=>(0,o.qy)`<wui-list-item
              @click=${()=>this.onExchangeClick(t)}
              chevron
              variant="image"
              imageSrc=${t.imageUrl}
              ?loading=${this.isLoading}
              ?disabled=${!this.amount}
            >
              <wui-text variant="paragraph-500" color="fg-200">
                Deposit from ${t.name}
              </wui-text>
            </wui-list-item>`)}
      </wui-flex>
    `}amountInputTemplate(){return(0,o.qy)`
      <wui-flex flexDirection="column" gap="s" .padding=${["0","s","s","s"]} class="amount-input-container">
        <wui-flex justifyContent="space-between">
          <wui-text variant="paragraph-500" color="fg-200">Asset</wui-text>
          <wui-chip-button
            data-testid="deposit-from-exchange-asset-button"
            text=${this.network?.nativeCurrency.symbol||""}
            imageSrc=${I.$.getNetworkImage(this.network)}
            size="sm"
            variant="gray"
            icon=${null}
          ></wui-chip-button>
        </wui-flex>
        <wui-flex flexDirection="column" alignItems="center" justifyContent="center">
          <wui-flex alignItems="center" gap="4xs">
            <wui-text variant="2xl-500" color="fg-200">${this.amount}</wui-text>
            <wui-text variant="paragraph-500" color="fg-200">USD</wui-text>
          </wui-flex>
          ${this.tokenAmountTemplate()}
          </wui-flex>
          <wui-flex justifyContent="space-between" gap="xs">
            ${A.map(t=>(0,o.qy)`<wui-button @click=${()=>this.onPresetAmountClick(t)} variant=${this.amount===t?"accent":"shade"} size="sm" fullWidth>$${t}</wui-button>`)}
          </wui-flex>
        </wui-flex>
      </wui-flex>
    `}tokenAmountTemplate(){return this.priceLoading?(0,o.qy)`<wui-shimmer
        width="65px"
        height="20px"
        borderRadius="xxs"
        variant="light"
      ></wui-shimmer>`:(0,o.qy)`
      <wui-text variant="paragraph-500" color="fg-200">
        ${this.tokenAmount} ${this.network?.nativeCurrency.symbol}
      </wui-text>
    `}async onExchangeClick(t){this.amount&&await z.handlePayWithExchange(t.id)}handlePaymentInProgress(){this.isPaymentInProgress&&this.currentPayment?.exchangeId&&this.currentPayment?.sessionId&&this.paymentId&&(k.P.showLoading("Deposit in progress..."),E.I.replace("Account"),z.waitUntilComplete({exchangeId:this.currentPayment.exchangeId,sessionId:this.currentPayment.sessionId,paymentId:this.paymentId}).then(t=>{"SUCCESS"===t.status?k.P.showSuccess("Deposit completed"):"FAILED"===t.status&&k.P.showError("Deposit failed")}))}onPresetAmountClick(t){z.setAmount(t)}};R.styles=S,j([(0,r.wk)()],R.prototype,"network",void 0),j([(0,r.wk)()],R.prototype,"exchanges",void 0),j([(0,r.wk)()],R.prototype,"isLoading",void 0),j([(0,r.wk)()],R.prototype,"amount",void 0),j([(0,r.wk)()],R.prototype,"tokenAmount",void 0),j([(0,r.wk)()],R.prototype,"priceLoading",void 0),j([(0,r.wk)()],R.prototype,"isPaymentInProgress",void 0),j([(0,r.wk)()],R.prototype,"currentPayment",void 0),j([(0,r.wk)()],R.prototype,"paymentId",void 0),R=j([(0,C.EM)("w3m-deposit-from-exchange-view")],R)},63270:(t,e,a)=>{var o=a(88088),r=a(53227),i=a(69428);let s=(0,o.AH)`
  :host {
    display: block;
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-005);
    background: linear-gradient(
      120deg,
      var(--wui-color-bg-200) 5%,
      var(--wui-color-bg-200) 48%,
      var(--wui-color-bg-300) 55%,
      var(--wui-color-bg-300) 60%,
      var(--wui-color-bg-300) calc(60% + 10px),
      var(--wui-color-bg-200) calc(60% + 12px),
      var(--wui-color-bg-200) 100%
    );
    background-size: 250%;
    animation: shimmer 3s linear infinite reverse;
  }

  :host([variant='light']) {
    background: linear-gradient(
      120deg,
      var(--wui-color-bg-150) 5%,
      var(--wui-color-bg-150) 48%,
      var(--wui-color-bg-200) 55%,
      var(--wui-color-bg-200) 60%,
      var(--wui-color-bg-200) calc(60% + 10px),
      var(--wui-color-bg-150) calc(60% + 12px),
      var(--wui-color-bg-150) 100%
    );
    background-size: 250%;
  }

  @keyframes shimmer {
    from {
      background-position: -250% 0;
    }
    to {
      background-position: 250% 0;
    }
  }
`;var n=function(t,e,a,o){var r,i=arguments.length,s=i<3?e:null===o?o=Object.getOwnPropertyDescriptor(e,a):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,a,o);else for(var n=t.length-1;n>=0;n--)(r=t[n])&&(s=(i<3?r(s):i>3?r(e,a,s):r(e,a))||s);return i>3&&s&&Object.defineProperty(e,a,s),s};let c=class extends o.WF{constructor(){super(...arguments),this.width="",this.height="",this.borderRadius="m",this.variant="default"}render(){return this.style.cssText=`
      width: ${this.width};
      height: ${this.height};
      border-radius: clamp(0px,var(--wui-border-radius-${this.borderRadius}), 40px);
    `,(0,o.qy)`<slot></slot>`}};c.styles=[s],n([(0,r.MZ)()],c.prototype,"width",void 0),n([(0,r.MZ)()],c.prototype,"height",void 0),n([(0,r.MZ)()],c.prototype,"borderRadius",void 0),n([(0,r.MZ)()],c.prototype,"variant",void 0),c=n([(0,i.E)("wui-shimmer")],c)},66449:(t,e,a)=>{a(82989)},82989:(t,e,a)=>{var o=a(88088),r=a(53227);a(84915),a(12936);var i=a(94895),s=a(69428);let n=(0,o.AH)`
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
`;var c=function(t,e,a,o){var r,i=arguments.length,s=i<3?e:null===o?o=Object.getOwnPropertyDescriptor(e,a):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,a,o);else for(var n=t.length-1;n>=0;n--)(r=t[n])&&(s=(i<3?r(s):i>3?r(e,a,s):r(e,a))||s);return i>3&&s&&Object.defineProperty(e,a,s),s};let l={main:"inverse-100",inverse:"inverse-000",accent:"accent-100","accent-error":"error-100","accent-success":"success-100",neutral:"fg-100",disabled:"gray-glass-020",shade:"fg-100"},u={lg:"paragraph-600",md:"small-600",sm:"small-600",xs:"tiny-600"},d={lg:"md",md:"md",sm:"sm",xs:"sm"},p=class extends o.WF{constructor(){super(...arguments),this.size="lg",this.disabled=!1,this.fullWidth=!1,this.loading=!1,this.variant="main",this.hasIconLeft=!1,this.hasIconRight=!1,this.borderRadius="m"}render(){this.style.cssText=`
    --local-width: ${this.fullWidth?"100%":"auto"};
    --local-opacity-100: ${+!this.loading};
    --local-opacity-000: ${+!!this.loading};
    --local-border-radius: var(--wui-border-radius-${this.borderRadius});
    `;let t=this.textVariant??u[this.size];return(0,o.qy)`
      <button
        data-variant=${this.variant}
        data-icon-left=${this.hasIconLeft}
        data-icon-right=${this.hasIconRight}
        data-size=${this.size}
        ?disabled=${this.disabled}
      >
        ${this.loadingTemplate()}
        <slot name="iconLeft" @slotchange=${()=>this.handleSlotLeftChange()}></slot>
        <wui-text variant=${t} color="inherit">
          <slot></slot>
        </wui-text>
        <slot name="iconRight" @slotchange=${()=>this.handleSlotRightChange()}></slot>
      </button>
    `}handleSlotLeftChange(){this.hasIconLeft=!0}handleSlotRightChange(){this.hasIconRight=!0}loadingTemplate(){if(this.loading){let t=d[this.size],e=this.disabled?l.disabled:l[this.variant];return(0,o.qy)`<wui-loading-spinner color=${e} size=${t}></wui-loading-spinner>`}return(0,o.qy)``}};p.styles=[i.W5,i.fD,n],c([(0,r.MZ)()],p.prototype,"size",void 0),c([(0,r.MZ)({type:Boolean})],p.prototype,"disabled",void 0),c([(0,r.MZ)({type:Boolean})],p.prototype,"fullWidth",void 0),c([(0,r.MZ)({type:Boolean})],p.prototype,"loading",void 0),c([(0,r.MZ)()],p.prototype,"variant",void 0),c([(0,r.MZ)({type:Boolean})],p.prototype,"hasIconLeft",void 0),c([(0,r.MZ)({type:Boolean})],p.prototype,"hasIconRight",void 0),c([(0,r.MZ)()],p.prototype,"borderRadius",void 0),c([(0,r.MZ)()],p.prototype,"textVariant",void 0),p=c([(0,s.E)("wui-button")],p)},84915:(t,e,a)=>{var o=a(88088),r=a(53227),i=a(94895),s=a(69428);let n=(0,o.AH)`
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
`;var c=function(t,e,a,o){var r,i=arguments.length,s=i<3?e:null===o?o=Object.getOwnPropertyDescriptor(e,a):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,a,o);else for(var n=t.length-1;n>=0;n--)(r=t[n])&&(s=(i<3?r(s):i>3?r(e,a,s):r(e,a))||s);return i>3&&s&&Object.defineProperty(e,a,s),s};let l=class extends o.WF{constructor(){super(...arguments),this.color="accent-100",this.size="lg"}render(){return this.style.cssText=`--local-color: ${"inherit"===this.color?"inherit":`var(--wui-color-${this.color})`}`,this.dataset.size=this.size,(0,o.qy)`<svg viewBox="25 25 50 50">
      <circle r="20" cy="50" cx="50"></circle>
    </svg>`}};l.styles=[i.W5,n],c([(0,r.MZ)()],l.prototype,"color",void 0),c([(0,r.MZ)()],l.prototype,"size",void 0),l=c([(0,s.E)("wui-loading-spinner")],l)},88364:(t,e,a)=>{var o=a(88088),r=a(53227),i=a(94895),s=a(69428);let n=(0,o.AH)`
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
`;var c=function(t,e,a,o){var r,i=arguments.length,s=i<3?e:null===o?o=Object.getOwnPropertyDescriptor(e,a):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)s=Reflect.decorate(t,e,a,o);else for(var n=t.length-1;n>=0;n--)(r=t[n])&&(s=(i<3?r(s):i>3?r(e,a,s):r(e,a))||s);return i>3&&s&&Object.defineProperty(e,a,s),s};let l=class extends o.WF{constructor(){super(...arguments),this.src="./path/to/image.jpg",this.alt="Image",this.size=void 0,this.objectFit="cover"}render(){return this.objectFit&&(this.dataset.objectFit=this.objectFit),this.style.cssText=`
      --local-width: ${this.size?`var(--wui-icon-size-${this.size});`:"100%"};
      --local-height: ${this.size?`var(--wui-icon-size-${this.size});`:"100%"};
      `,(0,o.qy)`<img src=${this.src} alt=${this.alt} @error=${this.handleImageError} />`}handleImageError(){this.dispatchEvent(new CustomEvent("onLoadError",{bubbles:!0,composed:!0}))}};l.styles=[i.W5,i.ck,n],c([(0,r.MZ)()],l.prototype,"src",void 0),c([(0,r.MZ)()],l.prototype,"alt",void 0),c([(0,r.MZ)()],l.prototype,"size",void 0),c([(0,r.MZ)()],l.prototype,"objectFit",void 0),l=c([(0,s.E)("wui-image")],l)}}]);