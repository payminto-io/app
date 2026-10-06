"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[2057],{3356:(e,t,i)=>{var o=i(88088),a=i(53227),r=i(48164);i(19763),i(88364),i(12936),i(24621);var n=i(94895),s=i(69428);let l=(0,o.AH)`
  button {
    padding: 6.5px var(--wui-spacing-l) 6.5px var(--wui-spacing-xs);
    display: flex;
    justify-content: space-between;
    width: 100%;
    border-radius: var(--wui-border-radius-xs);
    background-color: var(--wui-color-gray-glass-002);
  }

  button[data-clickable='false'] {
    pointer-events: none;
    background-color: transparent;
  }

  wui-image,
  wui-icon {
    width: var(--wui-spacing-3xl);
    height: var(--wui-spacing-3xl);
  }

  wui-image {
    border-radius: var(--wui-border-radius-3xl);
  }
`;var c=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let d=class extends o.WF{constructor(){super(...arguments),this.tokenName="",this.tokenImageUrl="",this.tokenValue=0,this.tokenAmount="0.0",this.tokenCurrency="",this.clickable=!1}render(){return(0,o.qy)`
      <button data-clickable=${String(this.clickable)}>
        <wui-flex gap="s" alignItems="center">
          ${this.visualTemplate()}
          <wui-flex flexDirection="column" justifyContent="spaceBetween">
            <wui-text variant="paragraph-500" color="fg-100">${this.tokenName}</wui-text>
            <wui-text variant="small-400" color="fg-200">
              ${r.S.formatNumberToLocalString(this.tokenAmount,4)} ${this.tokenCurrency}
            </wui-text>
          </wui-flex>
        </wui-flex>
        <wui-text variant="paragraph-500" color="fg-100">$${this.tokenValue.toFixed(2)}</wui-text>
      </button>
    `}visualTemplate(){return this.tokenName&&this.tokenImageUrl?(0,o.qy)`<wui-image alt=${this.tokenName} src=${this.tokenImageUrl}></wui-image>`:(0,o.qy)`<wui-icon name="coinPlaceholder" color="fg-100"></wui-icon>`}};d.styles=[n.W5,n.fD,l],c([(0,a.MZ)()],d.prototype,"tokenName",void 0),c([(0,a.MZ)()],d.prototype,"tokenImageUrl",void 0),c([(0,a.MZ)({type:Number})],d.prototype,"tokenValue",void 0),c([(0,a.MZ)()],d.prototype,"tokenAmount",void 0),c([(0,a.MZ)()],d.prototype,"tokenCurrency",void 0),c([(0,a.MZ)({type:Boolean})],d.prototype,"clickable",void 0),d=c([(0,s.E)("wui-list-token")],d)},12933:(e,t,i)=>{i(80124)},18128:(e,t,i)=>{i(19690)},19690:(e,t,i)=>{var o=i(88088),a=i(53227);i(19763),i(88364),i(24621);var r=i(94895),n=i(69428);i(15642);let s=(0,o.AH)`
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
`;var l=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let c=class extends o.WF{constructor(){super(...arguments),this.size="md",this.name="",this.installed=!1,this.badgeSize="xs"}render(){let e="xxs";return e="lg"===this.size?"m":"md"===this.size?"xs":"xxs",this.style.cssText=`
       --local-border-radius: var(--wui-border-radius-${e});
       --local-size: var(--wui-wallet-image-size-${this.size});
   `,this.walletIcon&&(this.dataset.walletIcon=this.walletIcon),(0,o.qy)`
      <wui-flex justifyContent="center" alignItems="center"> ${this.templateVisual()} </wui-flex>
    `}templateVisual(){return this.imageSrc?(0,o.qy)`<wui-image src=${this.imageSrc} alt=${this.name}></wui-image>`:this.walletIcon?(0,o.qy)`<wui-icon
        data-parent-size="md"
        size="md"
        color="inherit"
        name=${this.walletIcon}
      ></wui-icon>`:(0,o.qy)`<wui-icon
      data-parent-size=${this.size}
      size="inherit"
      color="inherit"
      name="walletPlaceholder"
    ></wui-icon>`}};c.styles=[r.fD,r.W5,s],l([(0,a.MZ)()],c.prototype,"size",void 0),l([(0,a.MZ)()],c.prototype,"name",void 0),l([(0,a.MZ)()],c.prototype,"imageSrc",void 0),l([(0,a.MZ)()],c.prototype,"walletIcon",void 0),l([(0,a.MZ)({type:Boolean})],c.prototype,"installed",void 0),l([(0,a.MZ)()],c.prototype,"badgeSize",void 0),c=l([(0,n.E)("wui-wallet-image")],c)},22057:(e,t,i)=>{i.r(t),i.d(t,{AppKitAccountButton:()=>C,AppKitButton:()=>A,AppKitConnectButton:()=>P,AppKitNetworkButton:()=>B,W3mAccountButton:()=>k,W3mAccountSettingsView:()=>el,W3mAccountView:()=>eU,W3mAllWalletsView:()=>tR,W3mButton:()=>S,W3mChooseAccountNameView:()=>i5,W3mConnectButton:()=>M,W3mConnectView:()=>i$,W3mConnectWalletsView:()=>oo,W3mConnectingExternalView:()=>iP,W3mConnectingMultiChainView:()=>iD,W3mConnectingWcBasicView:()=>i0,W3mConnectingWcView:()=>iX,W3mDownloadsView:()=>i4,W3mGetWalletView:()=>i6,W3mNetworkButton:()=>Z,W3mNetworkSwitchView:()=>ol,W3mNetworksView:()=>og,W3mProfileWalletsView:()=>e7,W3mRouter:()=>H.J,W3mSIWXSignMessageView:()=>oq,W3mSwitchActiveChainView:()=>ov,W3mUnsupportedChainView:()=>o$,W3mWalletCompatibleNetworksView:()=>oW,W3mWhatIsANetworkView:()=>ox,W3mWhatIsAWalletView:()=>oe});var o=i(88088),a=i(53227),r=i(40859),n=i(19392),s=i(1024),l=i(76581),c=i(48789),d=i(34066),u=i(37056),h=i(24568),p=i(59306);i(88364),i(84915),i(12936),i(24621);var w=i(94895),g=i(21736),b=i(69428);i(26478),i(15642);let f=(0,o.AH)`
  :host {
    display: block;
  }

  button {
    border-radius: var(--wui-border-radius-3xl);
    background: var(--wui-color-gray-glass-002);
    display: flex;
    gap: var(--wui-spacing-xs);
    padding: var(--wui-spacing-3xs) var(--wui-spacing-xs) var(--wui-spacing-3xs)
      var(--wui-spacing-xs);
    border: 1px solid var(--wui-color-gray-glass-005);
  }

  button:disabled {
    background: var(--wui-color-gray-glass-015);
  }

  button:disabled > wui-text {
    color: var(--wui-color-gray-glass-015);
  }

  button:disabled > wui-flex > wui-text {
    color: var(--wui-color-gray-glass-015);
  }

  button:disabled > wui-image,
  button:disabled > wui-flex > wui-avatar {
    filter: grayscale(1);
  }

  button:has(wui-image) {
    padding: var(--wui-spacing-3xs) var(--wui-spacing-3xs) var(--wui-spacing-3xs)
      var(--wui-spacing-xs);
  }

  wui-text {
    color: var(--wui-color-fg-100);
  }

  wui-flex > wui-text {
    color: var(--wui-color-fg-200);
  }

  wui-image,
  wui-icon-box {
    border-radius: var(--wui-border-radius-3xl);
    width: 24px;
    height: 24px;
    box-shadow: 0 0 0 2px var(--wui-color-gray-glass-005);
  }

  wui-flex {
    border-radius: var(--wui-border-radius-3xl);
    border: 1px solid var(--wui-color-gray-glass-005);
    background: var(--wui-color-gray-glass-005);
    padding: 4px var(--wui-spacing-m) 4px var(--wui-spacing-xxs);
  }

  button.local-no-balance {
    border-radius: 0px;
    border: none;
    background: transparent;
  }

  wui-avatar {
    width: 20px;
    height: 20px;
    box-shadow: 0 0 0 2px var(--wui-color-accent-glass-010);
  }

  @media (max-width: 500px) {
    button {
      gap: 0px;
      padding: var(--wui-spacing-3xs) var(--wui-spacing-xs) !important;
      height: 32px;
    }
    wui-image,
    wui-icon-box,
    button > wui-text {
      visibility: hidden;
      width: 0px;
      height: 0px;
    }
    button {
      border-radius: 0px;
      border: none;
      background: transparent;
      padding: 0px;
    }
  }

  @media (hover: hover) and (pointer: fine) {
    button:hover:enabled > wui-flex > wui-text {
      color: var(--wui-color-fg-175);
    }

    button:active:enabled > wui-flex > wui-text {
      color: var(--wui-color-fg-175);
    }
  }
`;var m=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let v=class extends o.WF{constructor(){super(...arguments),this.networkSrc=void 0,this.avatarSrc=void 0,this.balance=void 0,this.isUnsupportedChain=void 0,this.disabled=!1,this.loading=!1,this.address="",this.profileName="",this.charsStart=4,this.charsEnd=6}render(){return(0,o.qy)`
      <button
        ?disabled=${this.disabled}
        class=${(0,r.J)(this.balance?void 0:"local-no-balance")}
      >
        ${this.balanceTemplate()}
        <wui-flex gap="xxs" alignItems="center">
          <wui-avatar
            .imageSrc=${this.avatarSrc}
            alt=${this.address}
            address=${this.address}
          ></wui-avatar>
          <wui-text variant="paragraph-600" color="inherit">
            ${this.address?g.Z.getTruncateString({string:this.profileName||this.address,charsStart:this.profileName?18:this.charsStart,charsEnd:this.profileName?0:this.charsEnd,truncate:this.profileName?"end":"middle"}):null}
          </wui-text>
        </wui-flex>
      </button>
    `}balanceTemplate(){if(this.isUnsupportedChain)return(0,o.qy)` <wui-icon-box
          size="sm"
          iconColor="error-100"
          backgroundColor="error-100"
          icon="warningCircle"
          data-testid="wui-account-button-unsupported-chain"
        ></wui-icon-box>
        <wui-text variant="paragraph-600" color="inherit"> Switch Network</wui-text>`;if(this.balance){let e=this.networkSrc?(0,o.qy)`<wui-image src=${this.networkSrc}></wui-image>`:(0,o.qy)`
            <wui-icon-box
              size="sm"
              iconColor="fg-200"
              backgroundColor="fg-300"
              icon="networkPlaceholder"
            ></wui-icon-box>
          `,t=this.loading?(0,o.qy)`<wui-loading-spinner size="md" color="fg-200"></wui-loading-spinner>`:(0,o.qy)`<wui-text variant="paragraph-600" color="inherit"> ${this.balance}</wui-text>`;return(0,o.qy)`${e} ${t}`}return null}};v.styles=[w.W5,w.fD,f],m([(0,a.MZ)()],v.prototype,"networkSrc",void 0),m([(0,a.MZ)()],v.prototype,"avatarSrc",void 0),m([(0,a.MZ)()],v.prototype,"balance",void 0),m([(0,a.MZ)({type:Boolean})],v.prototype,"isUnsupportedChain",void 0),m([(0,a.MZ)({type:Boolean})],v.prototype,"disabled",void 0),m([(0,a.MZ)({type:Boolean})],v.prototype,"loading",void 0),m([(0,a.MZ)()],v.prototype,"address",void 0),m([(0,a.MZ)()],v.prototype,"profileName",void 0),m([(0,a.MZ)()],v.prototype,"charsStart",void 0),m([(0,a.MZ)()],v.prototype,"charsEnd",void 0),v=m([(0,b.E)("wui-account-button")],v);var y=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};class x extends o.WF{constructor(){super(...arguments),this.unsubscribe=[],this.disabled=!1,this.balance="show",this.charsStart=4,this.charsEnd=6,this.namespace=void 0,this.isSupported=!!n.H.state.allowUnsupportedChain||!s.W.state.activeChain||s.W.checkIfSupportedNetwork(s.W.state.activeChain)}connectedCallback(){super.connectedCallback(),this.setAccountData(s.W.getAccountData(this.namespace)),this.setNetworkData(s.W.getNetworkData(this.namespace))}firstUpdated(){let e=this.namespace;e?this.unsubscribe.push(s.W.subscribeChainProp("accountState",e=>{this.setAccountData(e)},e),s.W.subscribeChainProp("networkState",t=>{this.setNetworkData(t),this.isSupported=s.W.checkIfSupportedNetwork(e,t?.caipNetwork?.caipNetworkId)},e)):this.unsubscribe.push(l.j.subscribeNetworkImages(()=>{this.networkImage=c.$.getNetworkImage(this.network)}),s.W.subscribeKey("activeCaipAddress",e=>{this.caipAddress=e}),d.U.subscribeKey("balance",e=>this.balanceVal=e),d.U.subscribeKey("balanceSymbol",e=>this.balanceSymbol=e),d.U.subscribeKey("profileName",e=>this.profileName=e),d.U.subscribeKey("profileImage",e=>this.profileImage=e),s.W.subscribeKey("activeCaipNetwork",e=>{this.network=e,this.networkImage=c.$.getNetworkImage(e),this.isSupported=!e?.chainNamespace||s.W.checkIfSupportedNetwork(e?.chainNamespace),this.fetchNetworkImage(e)}))}updated(){this.fetchNetworkImage(this.network)}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){if(!s.W.state.activeChain)return null;let e="show"===this.balance,t="string"!=typeof this.balanceVal;return(0,o.qy)`
      <wui-account-button
        .disabled=${!!this.disabled}
        .isUnsupportedChain=${!n.H.state.allowUnsupportedChain&&!this.isSupported}
        address=${(0,r.J)(u.w.getPlainAddress(this.caipAddress))}
        profileName=${(0,r.J)(this.profileName)}
        networkSrc=${(0,r.J)(this.networkImage)}
        avatarSrc=${(0,r.J)(this.profileImage)}
        balance=${e?u.w.formatBalance(this.balanceVal,this.balanceSymbol):""}
        @click=${this.onClick.bind(this)}
        data-testid=${`account-button${this.namespace?`-${this.namespace}`:""}`}
        .charsStart=${this.charsStart}
        .charsEnd=${this.charsEnd}
        ?loading=${t}
      >
      </wui-account-button>
    `}onClick(){this.isSupported||n.H.state.allowUnsupportedChain?h.W.open({namespace:this.namespace}):h.W.open({view:"UnsupportedChain"})}async fetchNetworkImage(e){e?.assets?.imageId&&(this.networkImage=await c.$.fetchNetworkImage(e?.assets?.imageId))}setAccountData(e){e&&(this.caipAddress=e.caipAddress,this.balanceVal=e.balance,this.balanceSymbol=e.balanceSymbol,this.profileName=e.profileName,this.profileImage=e.profileImage)}setNetworkData(e){e&&(this.network=e.caipNetwork,this.networkImage=c.$.getNetworkImage(e.caipNetwork))}}y([(0,a.MZ)({type:Boolean})],x.prototype,"disabled",void 0),y([(0,a.MZ)()],x.prototype,"balance",void 0),y([(0,a.MZ)()],x.prototype,"charsStart",void 0),y([(0,a.MZ)()],x.prototype,"charsEnd",void 0),y([(0,a.MZ)()],x.prototype,"namespace",void 0),y([(0,a.wk)()],x.prototype,"caipAddress",void 0),y([(0,a.wk)()],x.prototype,"balanceVal",void 0),y([(0,a.wk)()],x.prototype,"balanceSymbol",void 0),y([(0,a.wk)()],x.prototype,"profileName",void 0),y([(0,a.wk)()],x.prototype,"profileImage",void 0),y([(0,a.wk)()],x.prototype,"network",void 0),y([(0,a.wk)()],x.prototype,"networkImage",void 0),y([(0,a.wk)()],x.prototype,"isSupported",void 0);let k=class extends x{};k=y([(0,p.EM)("w3m-account-button")],k);let C=class extends x{};C=y([(0,p.EM)("appkit-account-button")],C);let $=(0,o.AH)`
  :host {
    display: block;
    width: max-content;
  }
`;var I=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};class E extends o.WF{constructor(){super(...arguments),this.unsubscribe=[],this.disabled=!1,this.balance=void 0,this.size=void 0,this.label=void 0,this.loadingLabel=void 0,this.charsStart=4,this.charsEnd=6,this.namespace=void 0}firstUpdated(){this.caipAddress=this.namespace?s.W.state.chains.get(this.namespace)?.accountState?.caipAddress:s.W.state.activeCaipAddress,this.namespace?this.unsubscribe.push(s.W.subscribeChainProp("accountState",e=>{this.caipAddress=e?.caipAddress},this.namespace)):this.unsubscribe.push(s.W.subscribeKey("activeCaipAddress",e=>this.caipAddress=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){return this.caipAddress?(0,o.qy)`
          <appkit-account-button
            .disabled=${!!this.disabled}
            balance=${(0,r.J)(this.balance)}
            .charsStart=${(0,r.J)(this.charsStart)}
            .charsEnd=${(0,r.J)(this.charsEnd)}
            namespace=${(0,r.J)(this.namespace)}
          >
          </appkit-account-button>
        `:(0,o.qy)`
          <appkit-connect-button
            size=${(0,r.J)(this.size)}
            label=${(0,r.J)(this.label)}
            loadingLabel=${(0,r.J)(this.loadingLabel)}
            namespace=${(0,r.J)(this.namespace)}
          ></appkit-connect-button>
        `}}E.styles=$,I([(0,a.MZ)({type:Boolean})],E.prototype,"disabled",void 0),I([(0,a.MZ)()],E.prototype,"balance",void 0),I([(0,a.MZ)()],E.prototype,"size",void 0),I([(0,a.MZ)()],E.prototype,"label",void 0),I([(0,a.MZ)()],E.prototype,"loadingLabel",void 0),I([(0,a.MZ)()],E.prototype,"charsStart",void 0),I([(0,a.MZ)()],E.prototype,"charsEnd",void 0),I([(0,a.MZ)()],E.prototype,"namespace",void 0),I([(0,a.wk)()],E.prototype,"caipAddress",void 0);let S=class extends E{};S=I([(0,p.EM)("w3m-button")],S);let A=class extends E{};A=I([(0,p.EM)("appkit-button")],A);let W=(0,o.AH)`
  :host {
    position: relative;
    display: block;
  }

  button {
    background: var(--wui-color-accent-100);
    border: 1px solid var(--wui-color-gray-glass-010);
    border-radius: var(--wui-border-radius-m);
    gap: var(--wui-spacing-xs);
  }

  button.loading {
    background: var(--wui-color-gray-glass-010);
    border: 1px solid var(--wui-color-gray-glass-010);
    pointer-events: none;
  }

  button:disabled {
    background-color: var(--wui-color-gray-glass-015);
    border: 1px solid var(--wui-color-gray-glass-010);
  }

  button:disabled > wui-text {
    color: var(--wui-color-gray-glass-015);
  }

  @media (hover: hover) and (pointer: fine) {
    button:hover:enabled {
      background-color: var(--wui-color-accent-090);
    }

    button:active:enabled {
      background-color: var(--wui-color-accent-080);
    }
  }

  button:focus-visible {
    border: 1px solid var(--wui-color-gray-glass-010);
    background-color: var(--wui-color-accent-090);
    -webkit-box-shadow: 0px 0px 0px 4px var(--wui-box-shadow-blue);
    -moz-box-shadow: 0px 0px 0px 4px var(--wui-box-shadow-blue);
    box-shadow: 0px 0px 0px 4px var(--wui-box-shadow-blue);
  }

  button[data-size='sm'] {
    padding: 6.75px 10px 7.25px;
  }

  ::slotted(*) {
    transition: opacity var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: opacity;
    opacity: var(--local-opacity-100);
  }

  button > wui-text {
    transition: opacity var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: opacity;
    opacity: var(--local-opacity-100);
    color: var(--wui-color-inverse-100);
  }

  button[data-size='md'] {
    padding: 9px var(--wui-spacing-l) 9px var(--wui-spacing-l);
  }

  button[data-size='md'] + wui-text {
    padding-left: var(--wui-spacing-3xs);
  }

  @media (max-width: 500px) {
    button[data-size='md'] {
      height: 32px;
      padding: 5px 12px;
    }

    button[data-size='md'] > wui-text > slot {
      font-size: 14px !important;
    }
  }

  wui-loading-spinner {
    width: 14px;
    height: 14px;
  }

  wui-loading-spinner::slotted(svg) {
    width: 10px !important;
    height: 10px !important;
  }

  button[data-size='sm'] > wui-loading-spinner {
    width: 12px;
    height: 12px;
  }
`;var T=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let R=class extends o.WF{constructor(){super(...arguments),this.size="md",this.loading=!1}render(){let e="md"===this.size?"paragraph-600":"small-600";return(0,o.qy)`
      <button data-size=${this.size} ?disabled=${this.loading}>
        ${this.loadingTemplate()}
        <wui-text variant=${e} color=${this.loading?"accent-100":"inherit"}>
          <slot></slot>
        </wui-text>
      </button>
    `}loadingTemplate(){return this.loading?(0,o.qy)`<wui-loading-spinner size=${this.size} color="accent-100"></wui-loading-spinner>`:null}};R.styles=[w.W5,w.fD,W],T([(0,a.MZ)()],R.prototype,"size",void 0),T([(0,a.MZ)({type:Boolean})],R.prototype,"loading",void 0),R=T([(0,b.E)("wui-connect-button")],R);var O=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};class N extends o.WF{constructor(){super(),this.unsubscribe=[],this.size="md",this.label="Connect Wallet",this.loadingLabel="Connecting...",this.open=h.W.state.open,this.loading=this.namespace?h.W.state.loadingNamespaceMap.get(this.namespace):h.W.state.loading,this.unsubscribe.push(h.W.subscribe(e=>{this.open=e.open,this.loading=this.namespace?e.loadingNamespaceMap.get(this.namespace):e.loading}))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){return(0,o.qy)`
      <wui-connect-button
        size=${(0,r.J)(this.size)}
        .loading=${this.loading}
        @click=${this.onClick.bind(this)}
        data-testid=${`connect-button${this.namespace?`-${this.namespace}`:""}`}
      >
        ${this.loading?this.loadingLabel:this.label}
      </wui-connect-button>
    `}onClick(){this.open?h.W.close():this.loading||h.W.open({view:"Connect",namespace:this.namespace})}}O([(0,a.MZ)()],N.prototype,"size",void 0),O([(0,a.MZ)()],N.prototype,"label",void 0),O([(0,a.MZ)()],N.prototype,"loadingLabel",void 0),O([(0,a.MZ)()],N.prototype,"namespace",void 0),O([(0,a.wk)()],N.prototype,"open",void 0),O([(0,a.wk)()],N.prototype,"loading",void 0);let M=class extends N{};M=O([(0,p.EM)("w3m-connect-button")],M);let P=class extends N{};P=O([(0,p.EM)("appkit-connect-button")],P);var j=i(43242);let q=(0,o.AH)`
  :host {
    display: block;
  }

  button {
    border-radius: var(--wui-border-radius-3xl);
    display: flex;
    gap: var(--wui-spacing-xs);
    padding: var(--wui-spacing-2xs) var(--wui-spacing-s) var(--wui-spacing-2xs)
      var(--wui-spacing-xs);
    border: 1px solid var(--wui-color-gray-glass-010);
    background-color: var(--wui-color-gray-glass-005);
    color: var(--wui-color-fg-100);
  }

  button:disabled {
    border: 1px solid var(--wui-color-gray-glass-005);
    background-color: var(--wui-color-gray-glass-015);
    color: var(--wui-color-gray-glass-015);
  }

  @media (hover: hover) and (pointer: fine) {
    button:hover:enabled {
      background-color: var(--wui-color-gray-glass-010);
    }

    button:active:enabled {
      background-color: var(--wui-color-gray-glass-015);
    }
  }

  wui-image,
  wui-icon-box {
    border-radius: var(--wui-border-radius-3xl);
    width: 24px;
    height: 24px;
    box-shadow: 0 0 0 2px var(--wui-color-gray-glass-005);
  }
`;var D=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let L=class extends o.WF{constructor(){super(...arguments),this.imageSrc=void 0,this.isUnsupportedChain=void 0,this.disabled=!1}render(){return(0,o.qy)`
      <button data-testid="wui-network-button" ?disabled=${this.disabled}>
        ${this.visualTemplate()}
        <wui-text variant="paragraph-600" color="inherit">
          <slot></slot>
        </wui-text>
      </button>
    `}visualTemplate(){return this.isUnsupportedChain?(0,o.qy)`
        <wui-icon-box
          size="sm"
          iconColor="error-100"
          backgroundColor="error-100"
          icon="warningCircle"
        ></wui-icon-box>
      `:this.imageSrc?(0,o.qy)`<wui-image src=${this.imageSrc}></wui-image>`:(0,o.qy)`
      <wui-icon-box
        size="sm"
        iconColor="inverse-100"
        backgroundColor="fg-100"
        icon="networkPlaceholder"
      ></wui-icon-box>
    `}};L.styles=[w.W5,w.fD,q],D([(0,a.MZ)()],L.prototype,"imageSrc",void 0),D([(0,a.MZ)({type:Boolean})],L.prototype,"isUnsupportedChain",void 0),D([(0,a.MZ)({type:Boolean})],L.prototype,"disabled",void 0),L=D([(0,b.E)("wui-network-button")],L);let U=(0,o.AH)`
  :host {
    display: block;
    width: max-content;
  }
`;var z=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};class F extends o.WF{constructor(){super(),this.unsubscribe=[],this.disabled=!1,this.network=s.W.state.activeCaipNetwork,this.networkImage=c.$.getNetworkImage(this.network),this.caipAddress=s.W.state.activeCaipAddress,this.loading=h.W.state.loading,this.isSupported=!!n.H.state.allowUnsupportedChain||!s.W.state.activeChain||s.W.checkIfSupportedNetwork(s.W.state.activeChain),this.unsubscribe.push(l.j.subscribeNetworkImages(()=>{this.networkImage=c.$.getNetworkImage(this.network)}),s.W.subscribeKey("activeCaipAddress",e=>{this.caipAddress=e}),s.W.subscribeKey("activeCaipNetwork",e=>{this.network=e,this.networkImage=c.$.getNetworkImage(e),this.isSupported=!e?.chainNamespace||s.W.checkIfSupportedNetwork(e.chainNamespace),c.$.fetchNetworkImage(e?.assets?.imageId)}),h.W.subscribeKey("loading",e=>this.loading=e))}firstUpdated(){c.$.fetchNetworkImage(this.network?.assets?.imageId)}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){let e=!this.network||s.W.checkIfSupportedNetwork(this.network.chainNamespace);return(0,o.qy)`
      <wui-network-button
        .disabled=${!!(this.disabled||this.loading)}
        .isUnsupportedChain=${!n.H.state.allowUnsupportedChain&&!e}
        imageSrc=${(0,r.J)(this.networkImage)}
        @click=${this.onClick.bind(this)}
        data-testid="w3m-network-button"
      >
        ${this.getLabel()}
        <slot></slot>
      </wui-network-button>
    `}getLabel(){return this.network?this.isSupported||n.H.state.allowUnsupportedChain?this.network.name:"Switch Network":this.label?this.label:this.caipAddress?"Unknown Network":"Select Network"}onClick(){this.loading||(j.E.sendEvent({type:"track",event:"CLICK_NETWORKS"}),h.W.open({view:"Networks"}))}}F.styles=U,z([(0,a.MZ)({type:Boolean})],F.prototype,"disabled",void 0),z([(0,a.MZ)({type:String})],F.prototype,"label",void 0),z([(0,a.wk)()],F.prototype,"network",void 0),z([(0,a.wk)()],F.prototype,"networkImage",void 0),z([(0,a.wk)()],F.prototype,"caipAddress",void 0),z([(0,a.wk)()],F.prototype,"loading",void 0),z([(0,a.wk)()],F.prototype,"isSupported",void 0);let Z=class extends F{};Z=z([(0,p.EM)("w3m-network-button")],Z);let B=class extends F{};B=z([(0,p.EM)("appkit-network-button")],B);var H=i(94796),_=i(34386),V=i(11446),K=i(29400),J=i(44963),G=i(33452),Y=i(44458),X=i(70106),Q=i(16025);i(74970),i(35787),i(61327),i(19763),i(82989);let ee=(0,o.AH)`
  :host {
    display: block;
  }

  button {
    width: 100%;
    display: block;
    padding-top: var(--wui-spacing-l);
    padding-bottom: var(--wui-spacing-l);
    padding-left: var(--wui-spacing-s);
    padding-right: var(--wui-spacing-2l);
    border-radius: var(--wui-border-radius-s);
    background-color: var(--wui-color-accent-glass-010);
  }

  button:hover {
    background-color: var(--wui-color-accent-glass-015) !important;
  }

  button:active {
    background-color: var(--wui-color-accent-glass-020) !important;
  }
`;var et=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ei=class extends o.WF{constructor(){super(...arguments),this.label="",this.description="",this.icon="wallet"}render(){return(0,o.qy)`
      <button>
        <wui-flex gap="m" alignItems="center" justifyContent="space-between">
          <wui-icon-box
            size="lg"
            iconcolor="accent-100"
            backgroundcolor="accent-100"
            icon=${this.icon}
            background="transparent"
          ></wui-icon-box>

          <wui-flex flexDirection="column" gap="3xs">
            <wui-text variant="paragraph-500" color="fg-100">${this.label}</wui-text>
            <wui-text variant="small-400" color="fg-200">${this.description}</wui-text>
          </wui-flex>

          <wui-icon size="md" color="fg-200" name="chevronRight"></wui-icon>
        </wui-flex>
      </button>
    `}};ei.styles=[w.W5,w.fD,ee],et([(0,a.MZ)()],ei.prototype,"label",void 0),et([(0,a.MZ)()],ei.prototype,"description",void 0),et([(0,a.MZ)()],ei.prototype,"icon",void 0),ei=et([(0,b.E)("wui-notice-card")],ei),i(3658);var eo=i(67066),ea=i(5820),er=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let en=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.socialProvider=ea.i.getConnectedSocialProvider(),this.socialUsername=ea.i.getConnectedSocialUsername(),this.namespace=s.W.state.activeChain,this.unsubscribe.push(s.W.subscribeKey("activeChain",e=>{this.namespace=e}))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){let e=V.a.getConnectorId(this.namespace),t=V.a.getAuthConnector();if(!t||e!==_.o.CONNECTOR_ID.AUTH)return this.style.cssText="display: none",null;let i=t.provider.getEmail()??"";return i||this.socialUsername?(0,o.qy)`
      <wui-list-item
        variant="icon"
        iconVariant="overlay"
        icon=${this.socialProvider??"mail"}
        iconSize=${this.socialProvider?"xxl":"sm"}
        data-testid="w3m-account-email-update"
        ?chevron=${!this.socialProvider}
        @click=${()=>{this.onGoToUpdateEmail(i,this.socialProvider)}}
      >
        <wui-text variant="paragraph-500" color="fg-100">${this.getAuthName(i)}</wui-text>
      </wui-list-item>
    `:(this.style.cssText="display: none",null)}onGoToUpdateEmail(e,t){t||Y.I.push("UpdateEmailWallet",{email:e,redirectView:"Account"})}getAuthName(e){return this.socialUsername?"discord"===this.socialProvider&&this.socialUsername.endsWith("0")?this.socialUsername.slice(0,-1):this.socialUsername:e.length>30?`${e.slice(0,-3)}...`:e}};er([(0,a.wk)()],en.prototype,"namespace",void 0),en=er([(0,p.EM)("w3m-account-auth-button")],en);var es=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let el=class extends o.WF{constructor(){super(),this.usubscribe=[],this.networkImages=l.j.state.networkImages,this.address=d.U.state.address,this.profileImage=d.U.state.profileImage,this.profileName=d.U.state.profileName,this.network=s.W.state.activeCaipNetwork,this.disconnecting=!1,this.loading=!1,this.switched=!1,this.text="",this.remoteFeatures=n.H.state.remoteFeatures,this.usubscribe.push(d.U.subscribe(e=>{e.address&&(this.address=e.address,this.profileImage=e.profileImage,this.profileName=e.profileName)}),s.W.subscribeKey("activeCaipNetwork",e=>{e?.id&&(this.network=e)}),n.H.subscribeKey("remoteFeatures",e=>{this.remoteFeatures=e}))}disconnectedCallback(){this.usubscribe.forEach(e=>e())}render(){if(!this.address)throw Error("w3m-account-settings-view: No account provided");let e=this.networkImages[this.network?.assets?.imageId??""];return(0,o.qy)`
      <wui-flex
        flexDirection="column"
        alignItems="center"
        gap="l"
        .padding=${["0","xl","m","xl"]}
      >
        <wui-avatar
          alt=${this.address}
          address=${this.address}
          imageSrc=${(0,r.J)(this.profileImage)}
          size="2lg"
        ></wui-avatar>
        <wui-flex flexDirection="column" alignItems="center">
          <wui-flex gap="3xs" alignItems="center" justifyContent="center">
            <wui-text variant="title-6-600" color="fg-100" data-testid="account-settings-address">
              ${p.Zv.getTruncateString({string:this.address,charsStart:4,charsEnd:6,truncate:"middle"})}
            </wui-text>
            <wui-icon-link
              size="md"
              icon="copy"
              iconColor="fg-200"
              @click=${this.onCopyAddress}
            ></wui-icon-link>
          </wui-flex>
        </wui-flex>
      </wui-flex>
      <wui-flex flexDirection="column" gap="m">
        <wui-flex flexDirection="column" gap="xs" .padding=${["0","l","m","l"]}>
          ${this.authCardTemplate()}
          <w3m-account-auth-button></w3m-account-auth-button>
          <wui-list-item
            .variant=${e?"image":"icon"}
            iconVariant="overlay"
            icon="networkPlaceholder"
            imageSrc=${(0,r.J)(e)}
            ?chevron=${this.isAllowedNetworkSwitch()}
            @click=${this.onNetworks.bind(this)}
            data-testid="account-switch-network-button"
          >
            <wui-text variant="paragraph-500" color="fg-100">
              ${this.network?.name??"Unknown"}
            </wui-text>
          </wui-list-item>
          ${this.togglePreferredAccountBtnTemplate()} ${this.chooseNameButtonTemplate()}
          <wui-list-item
            variant="icon"
            iconVariant="overlay"
            icon="disconnect"
            ?chevron=${!1}
            .loading=${this.disconnecting}
            @click=${this.onDisconnect.bind(this)}
            data-testid="disconnect-button"
          >
            <wui-text variant="paragraph-500" color="fg-200">Disconnect</wui-text>
          </wui-list-item>
        </wui-flex>
      </wui-flex>
    `}chooseNameButtonTemplate(){let e=this.network?.chainNamespace,t=V.a.getConnectorId(e),i=V.a.getAuthConnector();return s.W.checkIfNamesSupported()&&i&&t===_.o.CONNECTOR_ID.AUTH&&!this.profileName?(0,o.qy)`
      <wui-list-item
        variant="icon"
        iconVariant="overlay"
        icon="id"
        iconSize="sm"
        ?chevron=${!0}
        @click=${this.onChooseName.bind(this)}
        data-testid="account-choose-name-button"
      >
        <wui-text variant="paragraph-500" color="fg-100">Choose account name </wui-text>
      </wui-list-item>
    `:null}authCardTemplate(){let e=V.a.getConnectorId(this.network?.chainNamespace),t=V.a.getAuthConnector(),{origin:i}=location;return!t||e!==_.o.CONNECTOR_ID.AUTH||i.includes(K.oU.SECURE_SITE)?null:(0,o.qy)`
      <wui-notice-card
        @click=${this.onGoToUpgradeView.bind(this)}
        label="Upgrade your wallet"
        description="Transition to a self-custodial wallet"
        icon="wallet"
        data-testid="w3m-wallet-upgrade-card"
      ></wui-notice-card>
    `}isAllowedNetworkSwitch(){let e=s.W.getAllRequestedCaipNetworks(),t=!!e&&e.length>1,i=e?.find(({id:e})=>e===this.network?.id);return t||!i}onCopyAddress(){try{this.address&&(u.w.copyToClopboard(this.address),J.P.showSuccess("Address copied"))}catch{J.P.showError("Failed to copy")}}togglePreferredAccountBtnTemplate(){let e=this.network?.chainNamespace,t=s.W.checkIfSmartAccountEnabled(),i=V.a.getConnectorId(e);return V.a.getAuthConnector()&&i===_.o.CONNECTOR_ID.AUTH&&t?(this.switched||(this.text=(0,G.lj)(e)===eo.Vl.ACCOUNT_TYPES.SMART_ACCOUNT?"Switch to your EOA":"Switch to your Smart Account"),(0,o.qy)`
      <wui-list-item
        variant="icon"
        iconVariant="overlay"
        icon="swapHorizontalBold"
        iconSize="sm"
        ?chevron=${!0}
        ?loading=${this.loading}
        @click=${this.changePreferredAccountType.bind(this)}
        data-testid="account-toggle-preferred-account-type"
      >
        <wui-text variant="paragraph-500" color="fg-100">${this.text}</wui-text>
      </wui-list-item>
    `):null}onChooseName(){Y.I.push("ChooseAccountName")}async changePreferredAccountType(){let e=this.network?.chainNamespace,t=s.W.checkIfSmartAccountEnabled(),i=(0,G.lj)(e)!==eo.Vl.ACCOUNT_TYPES.SMART_ACCOUNT&&t?eo.Vl.ACCOUNT_TYPES.SMART_ACCOUNT:eo.Vl.ACCOUNT_TYPES.EOA;V.a.getAuthConnector()&&(this.loading=!0,await X.x.setPreferredAccountType(i,e),this.text=i===eo.Vl.ACCOUNT_TYPES.SMART_ACCOUNT?"Switch to your EOA":"Switch to your Smart Account",this.switched=!0,Q.R.resetSend(),this.loading=!1,this.requestUpdate())}onNetworks(){this.isAllowedNetworkSwitch()&&Y.I.push("Networks")}async onDisconnect(){try{this.disconnecting=!0;let e=this.network?.chainNamespace,t=X.x.getConnections(e).length>0,i=e&&V.a.state.activeConnectorIds[e],o=this.remoteFeatures?.multiWallet;await X.x.disconnect(o?{id:i,namespace:e}:{}),t&&o&&(Y.I.push("ProfileWallets"),J.P.showSuccess("Wallet deleted"))}catch{j.E.sendEvent({type:"track",event:"DISCONNECT_ERROR",properties:{message:"Failed to disconnect"}}),J.P.showError("Failed to disconnect")}finally{this.disconnecting=!1}}onGoToUpgradeView(){j.E.sendEvent({type:"track",event:"EMAIL_UPGRADE_FROM_MODAL"}),Y.I.push("UpgradeEmailWallet")}};es([(0,a.wk)()],el.prototype,"address",void 0),es([(0,a.wk)()],el.prototype,"profileImage",void 0),es([(0,a.wk)()],el.prototype,"profileName",void 0),es([(0,a.wk)()],el.prototype,"network",void 0),es([(0,a.wk)()],el.prototype,"disconnecting",void 0),es([(0,a.wk)()],el.prototype,"loading",void 0),es([(0,a.wk)()],el.prototype,"switched",void 0),es([(0,a.wk)()],el.prototype,"text",void 0),es([(0,a.wk)()],el.prototype,"remoteFeatures",void 0),el=es([(0,p.EM)("w3m-account-settings-view")],el),i(66449),i(94548);let ec=(0,o.AH)`
  :host {
    display: inline-flex;
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-3xl);
    padding: var(--wui-spacing-3xs);
    position: relative;
    height: 36px;
    min-height: 36px;
    overflow: hidden;
  }

  :host::before {
    content: '';
    position: absolute;
    pointer-events: none;
    top: 4px;
    left: 4px;
    display: block;
    width: var(--local-tab-width);
    height: 28px;
    border-radius: var(--wui-border-radius-3xl);
    background-color: var(--wui-color-gray-glass-002);
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-002);
    transform: translateX(calc(var(--local-tab) * var(--local-tab-width)));
    transition: transform var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: background-color, opacity;
  }

  :host([data-type='flex'])::before {
    left: 3px;
    transform: translateX(calc((var(--local-tab) * 34px) + (var(--local-tab) * 4px)));
  }

  :host([data-type='flex']) {
    display: flex;
    padding: 0px 0px 0px 12px;
    gap: 4px;
  }

  :host([data-type='flex']) > button > wui-text {
    position: absolute;
    left: 18px;
    opacity: 0;
  }

  button[data-active='true'] > wui-icon,
  button[data-active='true'] > wui-text {
    color: var(--wui-color-fg-100);
  }

  button[data-active='false'] > wui-icon,
  button[data-active='false'] > wui-text {
    color: var(--wui-color-fg-200);
  }

  button[data-active='true']:disabled,
  button[data-active='false']:disabled {
    background-color: transparent;
    opacity: 0.5;
    cursor: not-allowed;
  }

  button[data-active='true']:disabled > wui-text {
    color: var(--wui-color-fg-200);
  }

  button[data-active='false']:disabled > wui-text {
    color: var(--wui-color-fg-300);
  }

  button > wui-icon,
  button > wui-text {
    pointer-events: none;
    transition: color var(--wui-e ase-out-power-1) var(--wui-duration-md);
    will-change: color;
  }

  button {
    width: var(--local-tab-width);
    transition: background-color var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: background-color;
  }

  :host([data-type='flex']) > button {
    width: 34px;
    position: relative;
    display: flex;
    justify-content: flex-start;
  }

  button:hover:enabled,
  button:active:enabled {
    background-color: transparent !important;
  }

  button:hover:enabled > wui-icon,
  button:active:enabled > wui-icon {
    transition: all var(--wui-ease-out-power-1) var(--wui-duration-lg);
    color: var(--wui-color-fg-125);
  }

  button:hover:enabled > wui-text,
  button:active:enabled > wui-text {
    transition: all var(--wui-ease-out-power-1) var(--wui-duration-lg);
    color: var(--wui-color-fg-125);
  }

  button {
    border-radius: var(--wui-border-radius-3xl);
  }
`;var ed=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let eu=class extends o.WF{constructor(){super(...arguments),this.tabs=[],this.onTabChange=()=>null,this.buttons=[],this.disabled=!1,this.localTabWidth="100px",this.activeTab=0,this.isDense=!1}render(){return this.isDense=this.tabs.length>3,this.style.cssText=`
      --local-tab: ${this.activeTab};
      --local-tab-width: ${this.localTabWidth};
    `,this.dataset.type=this.isDense?"flex":"block",this.tabs.map((e,t)=>{let i=t===this.activeTab;return(0,o.qy)`
        <button
          ?disabled=${this.disabled}
          @click=${()=>this.onTabClick(t)}
          data-active=${i}
          data-testid="tab-${e.label?.toLowerCase()}"
        >
          ${this.iconTemplate(e)}
          <wui-text variant="small-600" color="inherit"> ${e.label} </wui-text>
        </button>
      `})}firstUpdated(){this.shadowRoot&&this.isDense&&(this.buttons=[...this.shadowRoot.querySelectorAll("button")],setTimeout(()=>{this.animateTabs(0,!0)},0))}iconTemplate(e){return e.icon?(0,o.qy)`<wui-icon size="xs" color="inherit" name=${e.icon}></wui-icon>`:null}onTabClick(e){this.buttons&&this.animateTabs(e,!1),this.activeTab=e,this.onTabChange(e)}animateTabs(e,t){let i=this.buttons[this.activeTab],o=this.buttons[e],a=i?.querySelector("wui-text"),r=o?.querySelector("wui-text"),n=o?.getBoundingClientRect(),s=r?.getBoundingClientRect();i&&a&&!t&&e!==this.activeTab&&(a.animate([{opacity:0}],{duration:50,easing:"ease",fill:"forwards"}),i.animate([{width:"34px"}],{duration:500,easing:"ease",fill:"forwards"})),o&&n&&s&&r&&(e!==this.activeTab||t)&&(this.localTabWidth=`${Math.round(n.width+s.width)+6}px`,o.animate([{width:`${n.width+s.width}px`}],{duration:500*!t,fill:"forwards",easing:"ease"}),r.animate([{opacity:1}],{duration:125*!t,delay:200*!t,fill:"forwards",easing:"ease"}))}};eu.styles=[w.W5,w.fD,ec],ed([(0,a.MZ)({type:Array})],eu.prototype,"tabs",void 0),ed([(0,a.MZ)()],eu.prototype,"onTabChange",void 0),ed([(0,a.MZ)({type:Array})],eu.prototype,"buttons",void 0),ed([(0,a.MZ)({type:Boolean})],eu.prototype,"disabled",void 0),ed([(0,a.MZ)()],eu.prototype,"localTabWidth",void 0),ed([(0,a.wk)()],eu.prototype,"activeTab",void 0),ed([(0,a.wk)()],eu.prototype,"isDense",void 0),eu=ed([(0,b.E)("wui-tabs")],eu),i(25739);let eh=(0,o.AH)`
  button {
    display: flex;
    align-items: center;
    padding: var(--wui-spacing-xxs);
    border-radius: var(--wui-border-radius-xxs);
    column-gap: var(--wui-spacing-xs);
  }

  wui-image,
  .icon-box {
    width: var(--wui-spacing-xxl);
    height: var(--wui-spacing-xxl);
    border-radius: var(--wui-border-radius-3xs);
  }

  wui-text {
    flex: 1;
  }

  .icon-box {
    position: relative;
  }

  .icon-box[data-active='true'] {
    background-color: var(--wui-color-gray-glass-005);
  }

  .circle {
    position: absolute;
    left: 16px;
    top: 15px;
    width: var(--wui-spacing-1xs);
    height: var(--wui-spacing-1xs);
    background-color: var(--wui-color-success-100);
    border: 2px solid var(--wui-color-modal-bg);
    border-radius: 50%;
  }
`;var ep=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ew=class extends o.WF{constructor(){super(...arguments),this.address="",this.profileName="",this.alt="",this.imageSrc="",this.icon=void 0,this.iconSize="md",this.loading=!1,this.charsStart=4,this.charsEnd=6}render(){return(0,o.qy)`
      <button>
        ${this.leftImageTemplate()} ${this.textTemplate()} ${this.rightImageTemplate()}
      </button>
    `}leftImageTemplate(){let e=this.icon?(0,o.qy)`<wui-icon
          size=${this.iconSize}
          color="fg-200"
          name=${this.icon}
          class="icon"
        ></wui-icon>`:(0,o.qy)`<wui-image src=${this.imageSrc} alt=${this.alt}></wui-image>`;return(0,o.qy)`
      <wui-flex
        alignItems="center"
        justifyContent="center"
        class="icon-box"
        data-active=${!!this.icon}
      >
        ${e}
        <wui-flex class="circle"></wui-flex>
      </wui-flex>
    `}textTemplate(){return(0,o.qy)`
      <wui-text variant="paragraph-500" color="fg-100">
        ${g.Z.getTruncateString({string:this.profileName||this.address,charsStart:this.profileName?16:this.charsStart,charsEnd:this.profileName?0:this.charsEnd,truncate:this.profileName?"end":"middle"})}
      </wui-text>
    `}rightImageTemplate(){return(0,o.qy)`<wui-icon name="chevronBottom" size="xs" color="fg-200"></wui-icon>`}};ew.styles=[w.W5,w.fD,eh],ep([(0,a.MZ)()],ew.prototype,"address",void 0),ep([(0,a.MZ)()],ew.prototype,"profileName",void 0),ep([(0,a.MZ)()],ew.prototype,"alt",void 0),ep([(0,a.MZ)()],ew.prototype,"imageSrc",void 0),ep([(0,a.MZ)()],ew.prototype,"icon",void 0),ep([(0,a.MZ)()],ew.prototype,"iconSize",void 0),ep([(0,a.MZ)({type:Boolean})],ew.prototype,"loading",void 0),ep([(0,a.MZ)({type:Number})],ew.prototype,"charsStart",void 0),ep([(0,a.MZ)({type:Number})],ew.prototype,"charsEnd",void 0),ew=ep([(0,b.E)("wui-wallet-switch")],ew);let eg=(0,o.AH)`
  wui-flex {
    width: 100%;
  }

  :host > wui-flex:first-child {
    transform: translateY(calc(var(--wui-spacing-xxs) * -1));
  }

  wui-icon-link {
    margin-right: calc(var(--wui-icon-box-size-md) * -1);
  }

  wui-notice-card {
    margin-bottom: var(--wui-spacing-3xs);
  }

  wui-list-item > wui-text {
    flex: 1;
  }

  w3m-transactions-view {
    max-height: 200px;
  }

  .tab-content-container {
    height: 300px;
    overflow-y: auto;
    overflow-x: hidden;
    scrollbar-width: none;
  }

  .tab-content-container::-webkit-scrollbar {
    display: none;
  }

  .account-button {
    width: auto;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--wui-spacing-s);
    height: 48px;
    padding: var(--wui-spacing-xs);
    padding-right: var(--wui-spacing-s);
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-002);
    background-color: var(--wui-color-gray-glass-002);
    border-radius: 24px;
    transition: background-color 0.2s linear;
  }

  .account-button:hover {
    background-color: var(--wui-color-gray-glass-005);
  }

  .avatar-container {
    position: relative;
  }

  wui-avatar.avatar {
    width: 32px;
    height: 32px;
    box-shadow: 0 0 0 2px var(--wui-color-gray-glass-005);
  }

  wui-wallet-switch {
    margin-top: var(--wui-spacing-xs);
  }

  wui-avatar.network-avatar {
    width: 16px;
    height: 16px;
    position: absolute;
    left: 100%;
    top: 100%;
    transform: translate(-75%, -75%);
    box-shadow: 0 0 0 2px var(--wui-color-gray-glass-005);
  }

  .account-links {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .account-links wui-flex {
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    background: red;
    align-items: center;
    justify-content: center;
    height: 48px;
    padding: 10px;
    flex: 1 0 0;
    border-radius: var(--XS, 16px);
    border: 1px solid var(--dark-accent-glass-010, rgba(71, 161, 255, 0.1));
    background: var(--dark-accent-glass-010, rgba(71, 161, 255, 0.1));
    transition:
      background-color var(--wui-ease-out-power-1) var(--wui-duration-md),
      opacity var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: background-color, opacity;
  }

  .account-links wui-flex:hover {
    background: var(--dark-accent-glass-015, rgba(71, 161, 255, 0.15));
  }

  .account-links wui-flex wui-icon {
    width: var(--S, 20px);
    height: var(--S, 20px);
  }

  .account-links wui-flex wui-icon svg path {
    stroke: #667dff;
  }
`;var eb=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ef=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.caipAddress=d.U.state.caipAddress,this.address=u.w.getPlainAddress(d.U.state.caipAddress),this.profileImage=d.U.state.profileImage,this.profileName=d.U.state.profileName,this.disconnecting=!1,this.balance=d.U.state.balance,this.balanceSymbol=d.U.state.balanceSymbol,this.features=n.H.state.features,this.remoteFeatures=n.H.state.remoteFeatures,this.namespace=s.W.state.activeChain,this.activeConnectorIds=V.a.state.activeConnectorIds,this.unsubscribe.push(d.U.subscribeKey("caipAddress",e=>{this.address=u.w.getPlainAddress(e),this.caipAddress=e}),d.U.subscribeKey("balance",e=>this.balance=e),d.U.subscribeKey("balanceSymbol",e=>this.balanceSymbol=e),d.U.subscribeKey("profileName",e=>this.profileName=e),d.U.subscribeKey("profileImage",e=>this.profileImage=e),n.H.subscribeKey("features",e=>this.features=e),n.H.subscribeKey("remoteFeatures",e=>this.remoteFeatures=e),V.a.subscribeKey("activeConnectorIds",e=>{this.activeConnectorIds=e}),s.W.subscribeKey("activeChain",e=>this.namespace=e),s.W.subscribeKey("activeCaipNetwork",e=>{e?.chainNamespace&&(this.namespace=e?.chainNamespace)}))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){if(!this.caipAddress||!this.namespace)return null;let e=this.activeConnectorIds[this.namespace],t=e?V.a.getConnectorById(e):void 0,i=c.$.getConnectorImage(t);return(0,o.qy)`<wui-flex
        flexDirection="column"
        .padding=${["0","xl","m","xl"]}
        alignItems="center"
        gap="s"
      >
        <wui-avatar
          alt=${(0,r.J)(this.caipAddress)}
          address=${(0,r.J)(u.w.getPlainAddress(this.caipAddress))}
          imageSrc=${(0,r.J)(null===this.profileImage?void 0:this.profileImage)}
          data-testid="single-account-avatar"
        ></wui-avatar>
        <wui-wallet-switch
          profileName=${this.profileName}
          address=${this.address}
          imageSrc=${i}
          alt=${t?.name}
          @click=${this.onGoToProfileWalletsView.bind(this)}
          data-testid="wui-wallet-switch"
        ></wui-wallet-switch>
        <wui-flex flexDirection="column" alignItems="center">
          <wui-text variant="paragraph-500" color="fg-200">
            ${u.w.formatBalance(this.balance,this.balanceSymbol)}
          </wui-text>
        </wui-flex>
        ${this.explorerBtnTemplate()}
      </wui-flex>

      <wui-flex flexDirection="column" gap="xs" .padding=${["0","s","s","s"]}>
        ${this.authCardTemplate()} <w3m-account-auth-button></w3m-account-auth-button>
        ${this.orderedFeaturesTemplate()} ${this.activityTemplate()}
        <wui-list-item
          variant="icon"
          iconVariant="overlay"
          icon="disconnect"
          ?chevron=${!1}
          .loading=${this.disconnecting}
          @click=${this.onDisconnect.bind(this)}
          data-testid="disconnect-button"
        >
          <wui-text variant="paragraph-500" color="fg-200">Disconnect</wui-text>
        </wui-list-item>
      </wui-flex>`}fundWalletTemplate(){if(!this.namespace)return null;let e=K.oU.ONRAMP_SUPPORTED_CHAIN_NAMESPACES.includes(this.namespace),t=this.remoteFeatures?.onramp&&e,i=!!this.features?.receive;return t||i?(0,o.qy)`
      <wui-list-item
        data-testid="w3m-account-default-fund-wallet-button"
        iconVariant="blue"
        icon="dollar"
        ?chevron=${!0}
        @click=${this.handleClickFundWallet.bind(this)}
      >
        <wui-text variant="paragraph-500" color="fg-100">Fund wallet</wui-text>
      </wui-list-item>
    `:null}orderedFeaturesTemplate(){return(this.features?.walletFeaturesOrder||K.oU.DEFAULT_FEATURES.walletFeaturesOrder).map(e=>{switch(e){case"onramp":return this.fundWalletTemplate();case"swaps":return this.swapsTemplate();case"send":return this.sendTemplate();default:return null}})}activityTemplate(){return this.namespace&&this.remoteFeatures?.activity&&K.oU.ACTIVITY_ENABLED_CHAIN_NAMESPACES.includes(this.namespace)?(0,o.qy)` <wui-list-item
          iconVariant="blue"
          icon="clock"
          iconSize="sm"
          ?chevron=${!0}
          @click=${this.onTransactions.bind(this)}
          data-testid="w3m-account-default-activity-button"
        >
          <wui-text variant="paragraph-500" color="fg-100">Activity</wui-text>
        </wui-list-item>`:null}swapsTemplate(){let e=this.remoteFeatures?.swaps,t=s.W.state.activeChain===_.o.CHAIN.EVM;return e&&t?(0,o.qy)`
      <wui-list-item
        iconVariant="blue"
        icon="recycleHorizontal"
        ?chevron=${!0}
        @click=${this.handleClickSwap.bind(this)}
        data-testid="w3m-account-default-swaps-button"
      >
        <wui-text variant="paragraph-500" color="fg-100">Swap</wui-text>
      </wui-list-item>
    `:null}sendTemplate(){let e=this.features?.send,t=s.W.state.activeChain;if(!t)throw Error("SendController:sendTemplate - namespace is required");let i=K.oU.SEND_SUPPORTED_NAMESPACES.includes(t);return e&&i?(0,o.qy)`
      <wui-list-item
        iconVariant="blue"
        icon="send"
        ?chevron=${!0}
        @click=${this.handleClickSend.bind(this)}
        data-testid="w3m-account-default-send-button"
      >
        <wui-text variant="paragraph-500" color="fg-100">Send</wui-text>
      </wui-list-item>
    `:null}authCardTemplate(){let e=s.W.state.activeChain;if(!e)throw Error("AuthCardTemplate:authCardTemplate - namespace is required");let t=V.a.getConnectorId(e),i=V.a.getAuthConnector(),{origin:a}=location;return!i||t!==_.o.CONNECTOR_ID.AUTH||a.includes(K.oU.SECURE_SITE)?null:(0,o.qy)`
      <wui-notice-card
        @click=${this.onGoToUpgradeView.bind(this)}
        label="Upgrade your wallet"
        description="Transition to a self-custodial wallet"
        icon="wallet"
        data-testid="w3m-wallet-upgrade-card"
      ></wui-notice-card>
    `}handleClickFundWallet(){Y.I.push("FundWallet")}handleClickSwap(){Y.I.push("Swap")}handleClickSend(){Y.I.push("WalletSend")}explorerBtnTemplate(){return d.U.state.addressExplorerUrl?(0,o.qy)`
      <wui-button size="md" variant="neutral" @click=${this.onExplorer.bind(this)}>
        <wui-icon size="sm" color="inherit" slot="iconLeft" name="compass"></wui-icon>
        Block Explorer
        <wui-icon size="sm" color="inherit" slot="iconRight" name="externalLink"></wui-icon>
      </wui-button>
    `:null}onTransactions(){j.E.sendEvent({type:"track",event:"CLICK_TRANSACTIONS",properties:{isSmartAccount:(0,G.lj)(s.W.state.activeChain)===eo.Vl.ACCOUNT_TYPES.SMART_ACCOUNT}}),Y.I.push("Transactions")}async onDisconnect(){try{this.disconnecting=!0;let e=X.x.getConnections(this.namespace).length>0,t=this.namespace&&V.a.state.activeConnectorIds[this.namespace],i=this.remoteFeatures?.multiWallet;await X.x.disconnect(i?{id:t,namespace:this.namespace}:{}),e&&i&&(Y.I.push("ProfileWallets"),J.P.showSuccess("Wallet deleted"))}catch{j.E.sendEvent({type:"track",event:"DISCONNECT_ERROR",properties:{message:"Failed to disconnect"}}),J.P.showError("Failed to disconnect")}finally{this.disconnecting=!1}}onExplorer(){let e=d.U.state.addressExplorerUrl;e&&u.w.openHref(e,"_blank")}onGoToUpgradeView(){j.E.sendEvent({type:"track",event:"EMAIL_UPGRADE_FROM_MODAL"}),Y.I.push("UpgradeEmailWallet")}onGoToProfileWalletsView(){Y.I.push("ProfileWallets")}};ef.styles=eg,eb([(0,a.wk)()],ef.prototype,"caipAddress",void 0),eb([(0,a.wk)()],ef.prototype,"address",void 0),eb([(0,a.wk)()],ef.prototype,"profileImage",void 0),eb([(0,a.wk)()],ef.prototype,"profileName",void 0),eb([(0,a.wk)()],ef.prototype,"disconnecting",void 0),eb([(0,a.wk)()],ef.prototype,"balance",void 0),eb([(0,a.wk)()],ef.prototype,"balanceSymbol",void 0),eb([(0,a.wk)()],ef.prototype,"features",void 0),eb([(0,a.wk)()],ef.prototype,"remoteFeatures",void 0),eb([(0,a.wk)()],ef.prototype,"namespace",void 0),eb([(0,a.wk)()],ef.prototype,"activeConnectorIds",void 0),ef=eb([(0,p.EM)("w3m-account-default-widget")],ef);let em=(0,o.AH)`
  span {
    font-weight: 500;
    font-size: 40px;
    color: var(--wui-color-fg-100);
    line-height: 130%; /* 52px */
    letter-spacing: -1.6px;
    text-align: center;
  }

  .pennies {
    color: var(--wui-color-fg-200);
  }
`;var ev=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ey=class extends o.WF{constructor(){super(...arguments),this.dollars="0",this.pennies="00"}render(){return(0,o.qy)`<span>$${this.dollars}<span class="pennies">.${this.pennies}</span></span>`}};ey.styles=[w.W5,em],ev([(0,a.MZ)()],ey.prototype,"dollars",void 0),ev([(0,a.MZ)()],ey.prototype,"pennies",void 0),ey=ev([(0,b.E)("wui-balance")],ey),i(92730);let ex=(0,o.AH)`
  :host {
    display: block;
    padding: 9px var(--wui-spacing-s) 10px var(--wui-spacing-s);
    border-radius: var(--wui-border-radius-xxs);

    color: var(--wui-color-bg-100);
    position: relative;
  }

  :host([data-variant='shade']) {
    background-color: var(--wui-color-bg-150);
    border: 1px solid var(--wui-color-gray-glass-005);
  }

  :host([data-variant='shade']) > wui-text {
    color: var(--wui-color-fg-150);
  }

  :host([data-variant='fill']) {
    background-color: var(--wui-color-fg-100);
    border: none;
  }

  wui-icon {
    position: absolute;
    width: 12px !important;
    height: 4px !important;
  }

  wui-icon[data-placement='top'] {
    bottom: 0px;
    left: 50%;
    transform: translate(-50%, 95%);
  }

  wui-icon[data-placement='bottom'] {
    top: 0;
    left: 50%;
    transform: translate(-50%, -95%) rotate(180deg);
  }

  wui-icon[data-placement='right'] {
    top: 50%;
    left: 0;
    transform: translate(-65%, -50%) rotate(90deg);
  }

  wui-icon[data-placement='left'] {
    top: 50%;
    right: 0%;
    transform: translate(65%, -50%) rotate(270deg);
  }
`;var ek=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let eC=class extends o.WF{constructor(){super(...arguments),this.placement="top",this.variant="fill",this.message=""}render(){return this.dataset.variant=this.variant,(0,o.qy)`<wui-icon
        data-placement=${this.placement}
        color="fg-100"
        size="inherit"
        name=${"fill"===this.variant?"cursor":"cursorTransparent"}
      ></wui-icon>
      <wui-text color="inherit" variant="small-500">${this.message}</wui-text>`}};eC.styles=[w.W5,w.fD,ex],ek([(0,a.MZ)()],eC.prototype,"placement",void 0),ek([(0,a.MZ)()],eC.prototype,"variant",void 0),ek([(0,a.MZ)()],eC.prototype,"message",void 0),eC=ek([(0,b.E)("wui-tooltip")],eC);var e$=i(47316),eI=i(33526);i(58856);let eE=(0,o.AH)`
  :host {
    width: 100%;
    max-height: 280px;
    overflow: scroll;
    scrollbar-width: none;
  }

  :host::-webkit-scrollbar {
    display: none;
  }
`,eS=class extends o.WF{render(){return(0,o.qy)`<w3m-activity-list page="account"></w3m-activity-list>`}};eS.styles=eE,eS=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-account-activity-widget")],eS),i(37132),i(45667);let eA=(0,o.AH)`
  .contentContainer {
    height: 280px;
  }

  .contentContainer > wui-icon-box {
    width: 40px;
    height: 40px;
    border-radius: var(--wui-border-radius-xxs);
  }

  .contentContainer > .textContent {
    width: 65%;
  }
`,eW=class extends o.WF{render(){return(0,o.qy)`${this.nftTemplate()}`}nftTemplate(){return(0,o.qy)` <wui-flex
      class="contentContainer"
      alignItems="center"
      justifyContent="center"
      flexDirection="column"
      gap="l"
    >
      <wui-icon-box
        icon="wallet"
        size="inherit"
        iconColor="fg-200"
        backgroundColor="fg-200"
        iconSize="lg"
      ></wui-icon-box>
      <wui-flex
        class="textContent"
        gap="xs"
        flexDirection="column"
        justifyContent="center"
        flexDirection="column"
      >
        <wui-text
          variant="paragraph-500"
          align="center"
          color="fg-100"
          data-testid="nft-template-title"
          >Coming soon</wui-text
        >
        <wui-text
          variant="small-400"
          align="center"
          color="fg-200"
          data-testid="nft-template-description"
          >Stay tuned for our upcoming NFT feature</wui-text
        >
      </wui-flex>
      <wui-link @click=${this.onReceiveClick.bind(this)} data-testid="link-receive-funds"
        >Receive funds</wui-link
      >
    </wui-flex>`}onReceiveClick(){Y.I.push("WalletReceive")}};eW.styles=eA,eW=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-account-nfts-widget")],eW),i(54623);let eT=(0,o.AH)`
  button {
    width: 100%;
    display: flex;
    gap: var(--wui-spacing-s);
    align-items: center;
    justify-content: flex-start;
    padding: var(--wui-spacing-s) var(--wui-spacing-m) var(--wui-spacing-s) var(--wui-spacing-s);
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xs);
  }

  wui-icon-box {
    width: var(--wui-spacing-2xl);
    height: var(--wui-spacing-2xl);
  }

  wui-flex {
    width: auto;
  }
`;var eR=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let eO=class extends o.WF{constructor(){super(...arguments),this.icon="card",this.text="",this.description="",this.tag=void 0,this.iconBackgroundColor="accent-100",this.iconColor="accent-100",this.disabled=!1}render(){return(0,o.qy)`
      <button ?disabled=${this.disabled}>
        <wui-icon-box
          iconColor=${this.iconColor}
          backgroundColor=${this.iconBackgroundColor}
          size="inherit"
          icon=${this.icon}
          iconSize="md"
        ></wui-icon-box>
        <wui-flex flexDirection="column" justifyContent="spaceBetween">
          ${this.titleTemplate()}
          ${this.description?(0,o.qy)`<wui-text variant="small-400" color="fg-200"> ${this.description}</wui-text>`:null}</wui-flex
        >
      </button>
    `}titleTemplate(){return this.tag?(0,o.qy)` <wui-flex alignItems="center" gap="xxs"
        ><wui-text variant="paragraph-500" color="fg-100">${this.text}</wui-text
        ><wui-tag tagType="main" size="md">${this.tag}</wui-tag>
      </wui-flex>`:(0,o.qy)`<wui-text variant="paragraph-500" color="fg-100">${this.text}</wui-text>`}};eO.styles=[w.W5,w.fD,eT],eR([(0,a.MZ)()],eO.prototype,"icon",void 0),eR([(0,a.MZ)()],eO.prototype,"text",void 0),eR([(0,a.MZ)()],eO.prototype,"description",void 0),eR([(0,a.MZ)()],eO.prototype,"tag",void 0),eR([(0,a.MZ)()],eO.prototype,"iconBackgroundColor",void 0),eR([(0,a.MZ)()],eO.prototype,"iconColor",void 0),eR([(0,a.MZ)({type:Boolean})],eO.prototype,"disabled",void 0),eO=eR([(0,b.E)("wui-list-description")],eO),i(3356);let eN=(0,o.AH)`
  :host {
    width: 100%;
  }

  wui-flex {
    width: 100%;
  }

  .contentContainer {
    max-height: 280px;
    overflow: scroll;
    scrollbar-width: none;
  }

  .contentContainer::-webkit-scrollbar {
    display: none;
  }
`;var eM=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let eP=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tokenBalance=d.U.state.tokenBalance,this.remoteFeatures=n.H.state.remoteFeatures,this.unsubscribe.push(d.U.subscribe(e=>{this.tokenBalance=e.tokenBalance}),n.H.subscribeKey("remoteFeatures",e=>{this.remoteFeatures=e}))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){return(0,o.qy)`${this.tokenTemplate()}`}tokenTemplate(){return this.tokenBalance&&this.tokenBalance?.length>0?(0,o.qy)`<wui-flex class="contentContainer" flexDirection="column" gap="xs">
        ${this.tokenItemTemplate()}
      </wui-flex>`:(0,o.qy)` <wui-flex flexDirection="column" gap="xs"
      >${this.onRampTemplate()}
      <wui-list-description
        @click=${this.onReceiveClick.bind(this)}
        text="Receive funds"
        description="Scan the QR code and receive funds"
        icon="qrCode"
        iconColor="fg-200"
        iconBackgroundColor="fg-200"
        data-testid="w3m-account-receive-button"
      ></wui-list-description
    ></wui-flex>`}onRampTemplate(){return this.remoteFeatures?.onramp?(0,o.qy)`<wui-list-description
        @click=${this.onBuyClick.bind(this)}
        text="Buy Crypto"
        description="Easy with card or bank account"
        icon="card"
        iconColor="success-100"
        iconBackgroundColor="success-100"
        tag="popular"
        data-testid="w3m-account-onramp-button"
      ></wui-list-description>`:(0,o.qy)``}tokenItemTemplate(){return this.tokenBalance?.map(e=>(0,o.qy)`<wui-list-token
          tokenName=${e.name}
          tokenImageUrl=${e.iconUrl}
          tokenAmount=${e.quantity.numeric}
          tokenValue=${e.value}
          tokenCurrency=${e.symbol}
        ></wui-list-token>`)}onReceiveClick(){Y.I.push("WalletReceive")}onBuyClick(){j.E.sendEvent({type:"track",event:"SELECT_BUY_CRYPTO",properties:{isSmartAccount:(0,G.lj)(s.W.state.activeChain)===eo.Vl.ACCOUNT_TYPES.SMART_ACCOUNT}}),Y.I.push("OnRampProviders")}};eP.styles=eN,eM([(0,a.wk)()],eP.prototype,"tokenBalance",void 0),eM([(0,a.wk)()],eP.prototype,"remoteFeatures",void 0),eP=eM([(0,p.EM)("w3m-account-tokens-widget")],eP),i(78920),i(74203);let ej=(0,o.AH)`
  wui-flex {
    width: 100%;
  }

  wui-promo {
    position: absolute;
    top: -32px;
  }

  wui-profile-button {
    margin-top: calc(-1 * var(--wui-spacing-2l));
  }

  wui-promo + wui-profile-button {
    margin-top: var(--wui-spacing-2l);
  }

  wui-tabs {
    width: 100%;
  }

  .contentContainer {
    height: 280px;
  }

  .contentContainer > wui-icon-box {
    width: 40px;
    height: 40px;
    border-radius: var(--wui-border-radius-xxs);
  }

  .contentContainer > .textContent {
    width: 65%;
  }
`;var eq=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let eD=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.address=d.U.state.address,this.profileName=d.U.state.profileName,this.network=s.W.state.activeCaipNetwork,this.currentTab=d.U.state.currentTab,this.tokenBalance=d.U.state.tokenBalance,this.features=n.H.state.features,this.namespace=s.W.state.activeChain,this.activeConnectorIds=V.a.state.activeConnectorIds,this.remoteFeatures=n.H.state.remoteFeatures,this.unsubscribe.push(d.U.subscribe(e=>{e.address?(this.address=e.address,this.profileName=e.profileName,this.currentTab=e.currentTab,this.tokenBalance=e.tokenBalance):h.W.close()}),V.a.subscribeKey("activeConnectorIds",e=>{this.activeConnectorIds=e}),s.W.subscribeKey("activeChain",e=>this.namespace=e),s.W.subscribeKey("activeCaipNetwork",e=>this.network=e),n.H.subscribeKey("features",e=>this.features=e),n.H.subscribeKey("remoteFeatures",e=>this.remoteFeatures=e)),this.watchSwapValues()}disconnectedCallback(){this.unsubscribe.forEach(e=>e()),clearInterval(this.watchTokenBalance)}firstUpdated(){d.U.fetchTokenBalance()}render(){if(!this.address)throw Error("w3m-account-view: No account provided");if(!this.namespace)return null;let e=this.activeConnectorIds[this.namespace],t=e?V.a.getConnectorById(e):void 0,{icon:i,iconSize:a}=this.getAuthData();return(0,o.qy)`<wui-flex
      flexDirection="column"
      .padding=${["0","xl","m","xl"]}
      alignItems="center"
      gap="m"
      data-testid="w3m-account-wallet-features-widget"
    >
      <wui-flex flexDirection="column" justifyContent="center" alignItems="center" gap="xs">
        <wui-wallet-switch
          profileName=${this.profileName}
          address=${this.address}
          icon=${i}
          iconSize=${a}
          alt=${t?.name}
          @click=${this.onGoToProfileWalletsView.bind(this)}
          data-testid="wui-wallet-switch"
        ></wui-wallet-switch>

        ${this.tokenBalanceTemplate()}
      </wui-flex>
      ${this.orderedWalletFeatures()} ${this.tabsTemplate()} ${this.listContentTemplate()}
    </wui-flex>`}orderedWalletFeatures(){let e=this.features?.walletFeaturesOrder||K.oU.DEFAULT_FEATURES.walletFeaturesOrder;if(e.every(e=>"send"===e||"receive"===e?!this.features?.[e]:"swaps"!==e&&"onramp"!==e||!this.remoteFeatures?.[e]))return null;let t=[...new Set(e.map(e=>"receive"===e||"onramp"===e?"fund":e))];return(0,o.qy)`<wui-flex gap="s">
      ${t.map(e=>{switch(e){case"fund":return this.fundWalletTemplate();case"swaps":return this.swapsTemplate();case"send":return this.sendTemplate();default:return null}})}
    </wui-flex>`}fundWalletTemplate(){let e=this.remoteFeatures?.onramp,t=this.features?.receive;return e||t?(0,o.qy)`
      <w3m-tooltip-trigger text="Fund wallet">
        <wui-icon-button
          data-testid="wallet-features-fund-wallet-button"
          @click=${this.onFundWalletClick.bind(this)}
          icon="dollar"
        ></wui-icon-button>
      </w3m-tooltip-trigger>
    `:null}swapsTemplate(){let e=this.remoteFeatures?.swaps,t=s.W.state.activeChain===_.o.CHAIN.EVM;return e&&t?(0,o.qy)`
      <w3m-tooltip-trigger text="Swap">
        <wui-icon-button
          data-testid="wallet-features-swaps-button"
          @click=${this.onSwapClick.bind(this)}
          icon="recycleHorizontal"
        >
        </wui-icon-button>
      </w3m-tooltip-trigger>
    `:null}sendTemplate(){let e=this.features?.send,t=s.W.state.activeChain,i=K.oU.SEND_SUPPORTED_NAMESPACES.includes(t);return e&&i?(0,o.qy)`
      <w3m-tooltip-trigger text="Send">
        <wui-icon-button
          data-testid="wallet-features-send-button"
          @click=${this.onSendClick.bind(this)}
          icon="send"
        ></wui-icon-button>
      </w3m-tooltip-trigger>
    `:null}watchSwapValues(){this.watchTokenBalance=setInterval(()=>d.U.fetchTokenBalance(e=>this.onTokenBalanceError(e)),1e4)}onTokenBalanceError(e){e instanceof Error&&e.cause instanceof Response&&e.cause.status===_.o.HTTP_STATUS_CODES.SERVICE_UNAVAILABLE&&clearInterval(this.watchTokenBalance)}listContentTemplate(){return 0===this.currentTab?(0,o.qy)`<w3m-account-tokens-widget></w3m-account-tokens-widget>`:1===this.currentTab?(0,o.qy)`<w3m-account-nfts-widget></w3m-account-nfts-widget>`:2===this.currentTab?(0,o.qy)`<w3m-account-activity-widget></w3m-account-activity-widget>`:(0,o.qy)`<w3m-account-tokens-widget></w3m-account-tokens-widget>`}tokenBalanceTemplate(){if(this.tokenBalance&&this.tokenBalance?.length>=0){let e=u.w.calculateBalance(this.tokenBalance),{dollars:t="0",pennies:i="00"}=u.w.formatTokenBalance(e);return(0,o.qy)`<wui-balance dollars=${t} pennies=${i}></wui-balance>`}return(0,o.qy)`<wui-balance dollars="0" pennies="00"></wui-balance>`}tabsTemplate(){let e=eI.y.getTabsByNamespace(s.W.state.activeChain);if(0===e.length)return null;let t=u.w.isMobile()&&window.innerWidth<430,i="104px";return i=t?`${(window.innerWidth-48)/e.length}px`:2===e.length?"156px":"104px",(0,o.qy)`<wui-tabs
      .onTabChange=${this.onTabChange.bind(this)}
      .activeTab=${this.currentTab}
      localTabWidth=${i}
      .tabs=${e}
    ></wui-tabs>`}onTabChange(e){d.U.setCurrentTab(e)}onFundWalletClick(){Y.I.push("FundWallet")}onSwapClick(){this.network?.caipNetworkId&&!K.oU.SWAP_SUPPORTED_NETWORKS.includes(this.network?.caipNetworkId)?Y.I.push("UnsupportedChain",{swapUnsupportedChain:!0}):(j.E.sendEvent({type:"track",event:"OPEN_SWAP",properties:{network:this.network?.caipNetworkId||"",isSmartAccount:(0,G.lj)(s.W.state.activeChain)===eo.Vl.ACCOUNT_TYPES.SMART_ACCOUNT}}),Y.I.push("Swap"))}getAuthData(){let e=ea.i.getConnectedSocialProvider(),t=ea.i.getConnectedSocialUsername(),i=V.a.getAuthConnector(),o=i?.provider.getEmail()??"";return{name:e$.g.getAuthName({email:o,socialUsername:t,socialProvider:e}),icon:e??"mail",iconSize:e?"xl":"md"}}onGoToProfileWalletsView(){Y.I.push("ProfileWallets")}onSendClick(){j.E.sendEvent({type:"track",event:"OPEN_SEND",properties:{network:this.network?.caipNetworkId||"",isSmartAccount:(0,G.lj)(s.W.state.activeChain)===eo.Vl.ACCOUNT_TYPES.SMART_ACCOUNT}}),Y.I.push("WalletSend")}};eD.styles=ej,eq([(0,a.wk)()],eD.prototype,"watchTokenBalance",void 0),eq([(0,a.wk)()],eD.prototype,"address",void 0),eq([(0,a.wk)()],eD.prototype,"profileName",void 0),eq([(0,a.wk)()],eD.prototype,"network",void 0),eq([(0,a.wk)()],eD.prototype,"currentTab",void 0),eq([(0,a.wk)()],eD.prototype,"tokenBalance",void 0),eq([(0,a.wk)()],eD.prototype,"features",void 0),eq([(0,a.wk)()],eD.prototype,"namespace",void 0),eq([(0,a.wk)()],eD.prototype,"activeConnectorIds",void 0),eq([(0,a.wk)()],eD.prototype,"remoteFeatures",void 0),eD=eq([(0,p.EM)("w3m-account-wallet-features-widget")],eD);var eL=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let eU=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.namespace=s.W.state.activeChain,this.unsubscribe.push(s.W.subscribeKey("activeChain",e=>{this.namespace=e}))}render(){if(!this.namespace)return null;let e=V.a.getConnectorId(this.namespace),t=V.a.getAuthConnector();return(0,o.qy)`
      ${t&&e===_.o.CONNECTOR_ID.AUTH?this.walletFeaturesTemplate():this.defaultTemplate()}
    `}walletFeaturesTemplate(){return(0,o.qy)`<w3m-account-wallet-features-widget></w3m-account-wallet-features-widget>`}defaultTemplate(){return(0,o.qy)`<w3m-account-default-widget></w3m-account-default-widget>`}};eL([(0,a.wk)()],eU.prototype,"namespace",void 0),eU=eL([(0,p.EM)("w3m-account-view")],eU);var ez=i(74897),eF=i(30840),eZ=i(22787);i(20903),i(19690);let eB=(0,o.AH)`
  wui-image {
    width: var(--wui-spacing-2xl);
    height: var(--wui-spacing-2xl);
    border-radius: var(--wui-border-radius-3xs);
  }

  wui-image,
  .icon-box {
    width: var(--wui-spacing-2xl);
    height: var(--wui-spacing-2xl);
    border-radius: var(--wui-border-radius-3xs);
  }

  wui-icon:not(.custom-icon, .icon-badge) {
    cursor: pointer;
  }

  .icon-box {
    position: relative;
    background-color: var(--wui-color-gray-glass-002);
  }

  .icon-badge {
    position: absolute;
    top: 18px;
    left: 23px;
    z-index: 3;
    background-color: var(--wui-color-gray-glass-005);
    border: 2px solid var(--wui-color-modal-bg);
    border-radius: 50%;
    padding: var(--wui-spacing-4xs);
  }
`;var eH=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let e_=class extends o.WF{constructor(){super(...arguments),this.address="",this.profileName="",this.content=[],this.alt="",this.imageSrc="",this.icon=void 0,this.iconSize="md",this.iconBadge=void 0,this.iconBadgeSize="md",this.buttonVariant="neutral",this.enableMoreButton=!1,this.charsStart=4,this.charsEnd=6}render(){return(0,o.qy)`
      <wui-flex flexDirection="column" rowGap="xs">
        ${this.topTemplate()} ${this.bottomTemplate()}
      </wui-flex>
    `}topTemplate(){return(0,o.qy)`
      <wui-flex alignItems="flex-start" justifyContent="space-between">
        ${this.imageOrIconTemplate()}
        <wui-icon-link
          iconColor="fg-200"
          size="sm"
          icon="copy"
          @click=${this.dispatchCopyEvent}
        ></wui-icon-link>
        <wui-icon-link
          iconColor="fg-200"
          size="sm"
          icon="externalLink"
          @click=${this.dispatchExternalLinkEvent}
        ></wui-icon-link>
        ${this.enableMoreButton?(0,o.qy)`<wui-icon-link
              iconColor="fg-200"
              size="sm"
              icon="threeDots"
              @click=${this.dispatchMoreButtonEvent}
              data-testid="wui-active-profile-wallet-item-more-button"
            ></wui-icon-link>`:null}
      </wui-flex>
    `}bottomTemplate(){return(0,o.qy)` <wui-flex flexDirection="column">${this.contentTemplate()}</wui-flex> `}imageOrIconTemplate(){return this.icon?(0,o.qy)`
        <wui-flex flexGrow="1" alignItems="center">
          <wui-flex alignItems="center" justifyContent="center" class="icon-box">
            <wui-icon
              size=${this.iconSize}
              color="fg-200"
              name=${this.icon}
              class="custom-icon"
            ></wui-icon>

            ${this.iconBadge?(0,o.qy)`<wui-icon
                  color="fg-175"
                  size=${this.iconBadgeSize}
                  name=${this.iconBadge}
                  class="icon-badge"
                ></wui-icon>`:null}
          </wui-flex>
        </wui-flex>
      `:(0,o.qy)`
      <wui-flex flexGrow="1" alignItems="center">
        <wui-image objectFit="contain" src=${this.imageSrc} alt=${this.alt}></wui-image>
      </wui-flex>
    `}contentTemplate(){return 0===this.content.length?null:(0,o.qy)`
      <wui-flex flexDirection="column" rowGap="s">
        ${this.content.map(e=>this.labelAndTagTemplate(e))}
      </wui-flex>
    `}labelAndTagTemplate({address:e,profileName:t,label:i,description:a,enableButton:r,buttonType:n,buttonLabel:s,buttonVariant:l,tagVariant:c,tagLabel:d,alignItems:u="flex-end"}){return(0,o.qy)`
      <wui-flex justifyContent="space-between" alignItems=${u} columnGap="3xs">
        <wui-flex flexDirection="column" rowGap="4xs">
          ${i?(0,o.qy)`<wui-text variant="micro-600" color="fg-200">${i}</wui-text>`:null}

          <wui-flex alignItems="center" columnGap="3xs">
            <wui-text variant="small-500" color="fg-100">
              ${g.Z.getTruncateString({string:t||e,charsStart:t?16:this.charsStart,charsEnd:t?0:this.charsEnd,truncate:t?"end":"middle"})}
            </wui-text>

            ${c&&d?(0,o.qy)`<wui-tag variant=${c} size="xs">${d}</wui-tag>`:null}
          </wui-flex>

          ${a?(0,o.qy)`<wui-text variant="tiny-500" color="fg-200">${a}</wui-text>`:null}
        </wui-flex>

        ${r?this.buttonTemplate({buttonType:n,buttonLabel:s,buttonVariant:l}):null}
      </wui-flex>
    `}buttonTemplate({buttonType:e,buttonLabel:t,buttonVariant:i}){return(0,o.qy)`
      <wui-button
        size="xs"
        variant=${i}
        @click=${"disconnect"===e?this.dispatchDisconnectEvent.bind(this):this.dispatchSwitchEvent.bind(this)}
        data-testid=${"disconnect"===e?"wui-active-profile-wallet-item-disconnect-button":"wui-active-profile-wallet-item-switch-button"}
      >
        ${t}
      </wui-button>
    `}dispatchDisconnectEvent(){this.dispatchEvent(new CustomEvent("disconnect",{bubbles:!0,composed:!0}))}dispatchSwitchEvent(){this.dispatchEvent(new CustomEvent("switch",{bubbles:!0,composed:!0}))}dispatchExternalLinkEvent(){this.dispatchEvent(new CustomEvent("externalLink",{bubbles:!0,composed:!0}))}dispatchMoreButtonEvent(){this.dispatchEvent(new CustomEvent("more",{bubbles:!0,composed:!0}))}dispatchCopyEvent(){this.dispatchEvent(new CustomEvent("copy",{bubbles:!0,composed:!0}))}};e_.styles=[w.W5,w.fD,eB],eH([(0,a.MZ)()],e_.prototype,"address",void 0),eH([(0,a.MZ)()],e_.prototype,"profileName",void 0),eH([(0,a.MZ)({type:Array})],e_.prototype,"content",void 0),eH([(0,a.MZ)()],e_.prototype,"alt",void 0),eH([(0,a.MZ)()],e_.prototype,"imageSrc",void 0),eH([(0,a.MZ)()],e_.prototype,"icon",void 0),eH([(0,a.MZ)()],e_.prototype,"iconSize",void 0),eH([(0,a.MZ)()],e_.prototype,"iconBadge",void 0),eH([(0,a.MZ)()],e_.prototype,"iconBadgeSize",void 0),eH([(0,a.MZ)()],e_.prototype,"buttonVariant",void 0),eH([(0,a.MZ)({type:Boolean})],e_.prototype,"enableMoreButton",void 0),eH([(0,a.MZ)({type:Number})],e_.prototype,"charsStart",void 0),eH([(0,a.MZ)({type:Number})],e_.prototype,"charsEnd",void 0),e_=eH([(0,b.E)("wui-active-profile-wallet-item")],e_);let eV=(0,o.AH)`
  wui-image,
  .icon-box {
    width: var(--wui-spacing-2xl);
    height: var(--wui-spacing-2xl);
    border-radius: var(--wui-border-radius-3xs);
  }

  .right-icon {
    cursor: pointer;
  }

  .icon-box {
    position: relative;
    background-color: var(--wui-color-gray-glass-002);
  }

  .icon-badge {
    position: absolute;
    top: 18px;
    left: 23px;
    z-index: 3;
    background-color: var(--wui-color-gray-glass-005);
    border: 2px solid var(--wui-color-modal-bg);
    border-radius: 50%;
    padding: var(--wui-spacing-4xs);
  }
`;var eK=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let eJ=class extends o.WF{constructor(){super(...arguments),this.address="",this.profileName="",this.alt="",this.buttonLabel="",this.buttonVariant="accent",this.imageSrc="",this.icon=void 0,this.iconSize="md",this.iconBadgeSize="md",this.rightIcon="off",this.rightIconSize="md",this.loading=!1,this.charsStart=4,this.charsEnd=6}render(){return(0,o.qy)`
      <wui-flex alignItems="center" columnGap="xs">
        ${this.imageOrIconTemplate()} ${this.labelAndDescriptionTemplate()}
        ${this.buttonActionTemplate()}
      </wui-flex>
    `}imageOrIconTemplate(){return this.icon?(0,o.qy)`
        <wui-flex alignItems="center" justifyContent="center" class="icon-box">
          <wui-flex alignItems="center" justifyContent="center" class="icon-box">
            <wui-icon
              size=${this.iconSize}
              color="fg-200"
              name=${this.icon}
              class="custom-icon"
            ></wui-icon>
            ${this.iconBadge?(0,o.qy)`<wui-icon
                  color="fg-175"
                  size=${this.iconBadgeSize}
                  name=${this.iconBadge}
                  class="icon-badge"
                ></wui-icon>`:null}
          </wui-flex>
        </wui-flex>
      `:(0,o.qy)`<wui-image objectFit="contain" src=${this.imageSrc} alt=${this.alt}></wui-image>`}labelAndDescriptionTemplate(){return(0,o.qy)`
      <wui-flex
        flexDirection="column"
        flexGrow="1"
        justifyContent="flex-start"
        alignItems="flex-start"
      >
        <wui-text variant="small-500" color="fg-100">
          ${g.Z.getTruncateString({string:this.profileName||this.address,charsStart:this.profileName?16:this.charsStart,charsEnd:this.profileName?0:this.charsEnd,truncate:this.profileName?"end":"middle"})}
        </wui-text>
      </wui-flex>
    `}buttonActionTemplate(){return(0,o.qy)`
      <wui-flex columnGap="3xs" alignItems="center" justifyContent="center">
        <wui-button
          size="xs"
          variant=${this.buttonVariant}
          .loading=${this.loading}
          @click=${this.handleButtonClick}
          data-testid="wui-inactive-profile-wallet-item-button"
        >
          ${this.buttonLabel}
        </wui-button>

        <wui-icon-link
          iconColor="fg-200"
          size=${this.rightIconSize}
          icon=${this.rightIcon}
          class="right-icon"
          @click=${this.handleIconClick}
        ></wui-icon-link>
      </wui-flex>
    `}handleButtonClick(){this.dispatchEvent(new CustomEvent("buttonClick",{bubbles:!0,composed:!0}))}handleIconClick(){this.dispatchEvent(new CustomEvent("iconClick",{bubbles:!0,composed:!0}))}};eJ.styles=[w.W5,w.fD,eV],eK([(0,a.MZ)()],eJ.prototype,"address",void 0),eK([(0,a.MZ)()],eJ.prototype,"profileName",void 0),eK([(0,a.MZ)()],eJ.prototype,"alt",void 0),eK([(0,a.MZ)()],eJ.prototype,"buttonLabel",void 0),eK([(0,a.MZ)()],eJ.prototype,"buttonVariant",void 0),eK([(0,a.MZ)()],eJ.prototype,"imageSrc",void 0),eK([(0,a.MZ)()],eJ.prototype,"icon",void 0),eK([(0,a.MZ)()],eJ.prototype,"iconSize",void 0),eK([(0,a.MZ)()],eJ.prototype,"iconBadge",void 0),eK([(0,a.MZ)()],eJ.prototype,"iconBadgeSize",void 0),eK([(0,a.MZ)()],eJ.prototype,"rightIcon",void 0),eK([(0,a.MZ)()],eJ.prototype,"rightIconSize",void 0),eK([(0,a.MZ)({type:Boolean})],eJ.prototype,"loading",void 0),eK([(0,a.MZ)({type:Number})],eJ.prototype,"charsStart",void 0),eK([(0,a.MZ)({type:Number})],eJ.prototype,"charsEnd",void 0),eJ=eK([(0,b.E)("wui-inactive-profile-wallet-item")],eJ),i(52589);var eG=i(60890);let eY={getAuthData(e){let t=e.connectorId===_.o.CONNECTOR_ID.AUTH;if(!t)return{isAuth:!1,icon:void 0,iconSize:void 0,name:void 0};let i=e?.auth?.name??ea.i.getConnectedSocialProvider(),o=e?.auth?.username??ea.i.getConnectedSocialUsername(),a=V.a.getAuthConnector(),r=a?.provider.getEmail()??"";return{isAuth:!0,icon:i??"mail",iconSize:i?"xl":"md",name:t?e$.g.getAuthName({email:r,socialUsername:o,socialProvider:i}):void 0}}},eX=(0,o.AH)`
  :host {
    --connect-scroll--top-opacity: 0;
    --connect-scroll--bottom-opacity: 0;
  }

  .balance-amount {
    flex: 1;
  }

  .wallet-list {
    scrollbar-width: none;
    overflow-y: scroll;
    overflow-x: hidden;
    transition: opacity var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: opacity;
    mask-image: linear-gradient(
      to bottom,
      rgba(0, 0, 0, calc(1 - var(--connect-scroll--top-opacity))) 0px,
      rgba(200, 200, 200, calc(1 - var(--connect-scroll--top-opacity))) 1px,
      black 40px,
      black calc(100% - 40px),
      rgba(155, 155, 155, calc(1 - var(--connect-scroll--bottom-opacity))) calc(100% - 1px),
      rgba(0, 0, 0, calc(1 - var(--connect-scroll--bottom-opacity))) 100%
    );
  }

  .active-wallets {
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xs);
  }

  .active-wallets-box {
    height: 330px;
  }

  .empty-wallet-list-box {
    height: 400px;
  }

  .empty-box {
    width: 100%;
    padding: var(--wui-spacing-l);
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xs);
  }

  wui-separator {
    margin: var(--wui-spacing-xs) 0 var(--wui-spacing-xs) 0;
  }

  .active-connection {
    padding: var(--wui-spacing-xs);
  }

  .recent-connection {
    padding: var(--wui-spacing-xs) 0 var(--wui-spacing-xs) 0;
  }

  @media (max-width: 430px) {
    .active-wallets-box,
    .empty-wallet-list-box {
      height: auto;
      max-height: clamp(360px, 470px, 80vh);
    }
  }
`;var eQ=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let e0=4,e3=6,e1="md",e2="lightbulb",e5=[0,1],e4={eip155:"ethereum",solana:"solana",bip122:"bitcoin"},e6=[{namespace:"eip155",icon:e4.eip155,label:"EVM"},{namespace:"solana",icon:e4.solana,label:"Solana"},{namespace:"bip122",icon:e4.bip122,label:"Bitcoin"}],e8={eip155:{title:"Add EVM Wallet",description:"Add your first EVM wallet"},solana:{title:"Add Solana Wallet",description:"Add your first Solana wallet"},bip122:{title:"Add Bitcoin Wallet",description:"Add your first Bitcoin wallet"}},e7=class extends o.WF{constructor(){super(),this.unsubscribers=[],this.currentTab=0,this.namespace=s.W.state.activeChain,this.namespaces=Array.from(s.W.state.chains.keys()),this.caipAddress=void 0,this.profileName=void 0,this.activeConnectorIds=V.a.state.activeConnectorIds,this.lastSelectedAddress="",this.lastSelectedConnectorId="",this.isSwitching=!1,this.caipNetwork=s.W.state.activeCaipNetwork,this.user=d.U.state.user,this.remoteFeatures=n.H.state.remoteFeatures,this.tabWidth="",this.currentTab=this.namespace?this.namespaces.indexOf(this.namespace):0,this.caipAddress=s.W.getAccountData(this.namespace)?.caipAddress,this.profileName=s.W.getAccountData(this.namespace)?.profileName,this.unsubscribers.push(X.x.subscribeKey("connections",()=>this.onConnectionsChange()),X.x.subscribeKey("recentConnections",()=>this.requestUpdate()),V.a.subscribeKey("activeConnectorIds",e=>{this.activeConnectorIds=e}),s.W.subscribeKey("activeCaipNetwork",e=>this.caipNetwork=e),d.U.subscribeKey("user",e=>this.user=e),n.H.subscribeKey("remoteFeatures",e=>this.remoteFeatures=e)),this.chainListener=s.W.subscribeChainProp("accountState",e=>{this.caipAddress=e?.caipAddress,this.profileName=e?.profileName},this.namespace)}disconnectedCallback(){this.unsubscribers.forEach(e=>e()),this.resizeObserver?.disconnect(),this.tabsResizeObserver?.disconnect(),this.removeScrollListener(),this.chainListener?.()}firstUpdated(){let e=this.shadowRoot?.querySelector(".wallet-list"),t=this.shadowRoot?.querySelector("wui-tabs");if(!e)return;let i=()=>this.updateScrollOpacity(e);if(requestAnimationFrame(i),e.addEventListener("scroll",i),this.resizeObserver=new ResizeObserver(i),this.resizeObserver.observe(e),i(),t){let e=()=>{let e=e6.filter(e=>this.namespaces.includes(e.namespace)).length;if(e>1){let t=this.getBoundingClientRect()?.width;this.tabWidth=`${(t-32-8)/e}px`,this.requestUpdate()}};this.tabsResizeObserver=new ResizeObserver(e),this.tabsResizeObserver.observe(this),e()}}render(){let e=this.namespace;if(!e)throw Error("Namespace is not set");return(0,o.qy)`
      <wui-flex flexDirection="column" .padding=${["0","l","l","l"]} gap="l">
        ${this.renderTabs()} ${this.renderHeader(e)} ${this.renderConnections(e)}
        ${this.renderAddConnectionButton(e)}
      </wui-flex>
    `}renderTabs(){let e=e6.filter(e=>this.namespaces.includes(e.namespace));return e.length>1?(0,o.qy)`
        <wui-tabs
          .onTabChange=${e=>this.handleTabChange(e)}
          .activeTab=${this.currentTab}
          localTabWidth=${this.tabWidth}
          .tabs=${e}
        ></wui-tabs>
      `:null}renderHeader(e){let t=this.getActiveConnections(e).flatMap(({accounts:e})=>e).length+ +!!this.caipAddress;return(0,o.qy)`
      <wui-flex alignItems="center" columnGap="3xs">
        <wui-icon
          name=${e4[e]??e4.eip155}
          size="lg"
        ></wui-icon>
        <wui-text color="fg-200" variant="small-400"
          >${t>1?"Wallets":"Wallet"}</wui-text
        >
        <wui-text
          color="fg-100"
          variant="small-400"
          class="balance-amount"
          data-testid="balance-amount"
        >
          ${t}
        </wui-text>
        <wui-link
          color="fg-200"
          @click=${()=>X.x.disconnect({namespace:e})}
          ?disabled=${!this.hasAnyConnections(e)}
          data-testid="disconnect-all-button"
        >
          Disconnect All
        </wui-link>
      </wui-flex>
    `}renderConnections(e){let t=this.hasAnyConnections(e);return(0,o.qy)`
      <wui-flex flexDirection="column" class=${(0,ez.H)({"wallet-list":!0,"active-wallets-box":t,"empty-wallet-list-box":!t})} rowGap="s">
        ${t?this.renderActiveConnections(e):this.renderEmptyState(e)}
      </wui-flex>
    `}renderActiveConnections(e){let t=this.getActiveConnections(e),i=this.activeConnectorIds[e],a=this.getPlainAddress();return(0,o.qy)`
      ${a||i||t.length>0?(0,o.qy)`<wui-flex
            flexDirection="column"
            .padding=${["l","0","xs","0"]}
            class="active-wallets"
          >
            ${this.renderActiveProfile(e)} ${this.renderActiveConnectionsList(e)}
          </wui-flex>`:null}
      ${this.renderRecentConnections(e)}
    `}renderActiveProfile(e){let t=this.activeConnectorIds[e];if(!t)return null;let{connections:i}=eZ.b.getConnectionsData(e),a=V.a.getConnectorById(t),r=c.$.getConnectorImage(a),n=this.getPlainAddress();if(!n)return null;let s=e===_.o.CHAIN.BITCOIN,l=eY.getAuthData({connectorId:t,accounts:[]}),d=this.getActiveConnections(e).flatMap(e=>e.accounts).length>0,u=i.find(e=>e.connectorId===t),h=u?.accounts.filter(e=>!eG.y.isLowerCaseMatch(e.address,n));return(0,o.qy)`
      <wui-flex flexDirection="column" .padding=${["0","l","0","l"]}>
        <wui-active-profile-wallet-item
          address=${n}
          alt=${a?.name}
          .content=${this.getProfileContent({address:n,connections:i,connectorId:t,namespace:e})}
          .charsStart=${e0}
          .charsEnd=${e3}
          .icon=${l.icon}
          .iconSize=${l.iconSize}
          .iconBadge=${this.isSmartAccount(n)?e2:void 0}
          .iconBadgeSize=${this.isSmartAccount(n)?e1:void 0}
          imageSrc=${r}
          ?enableMoreButton=${l.isAuth}
          @copy=${()=>this.handleCopyAddress(n)}
          @disconnect=${()=>this.handleDisconnect(e,{id:t})}
          @switch=${()=>{s&&u&&h?.[0]&&this.handleSwitchWallet(u,h[0].address,e)}}
          @externalLink=${()=>this.handleExternalLink(n)}
          @more=${()=>this.handleMore()}
          data-testid="wui-active-profile-wallet-item"
        ></wui-active-profile-wallet-item>
        ${d?(0,o.qy)`<wui-separator></wui-separator>`:null}
      </wui-flex>
    `}renderActiveConnectionsList(e){let t=this.getActiveConnections(e);return 0===t.length?null:(0,o.qy)`
      <wui-flex flexDirection="column" .padding=${["0","xs","0","xs"]}>
        ${this.renderConnectionList(t,!1,e)}
      </wui-flex>
    `}renderRecentConnections(e){let{recentConnections:t}=eZ.b.getConnectionsData(e);return 0===t.flatMap(e=>e.accounts).length?null:(0,o.qy)`
      <wui-flex flexDirection="column" .padding=${["0","xs","0","xs"]} rowGap="xs">
        <wui-text color="fg-200" variant="micro-500" data-testid="recently-connected-text"
          >RECENTLY CONNECTED</wui-text
        >
        <wui-flex flexDirection="column" .padding=${["0","xs","0","xs"]}>
          ${this.renderConnectionList(t,!0,e)}
        </wui-flex>
      </wui-flex>
    `}renderConnectionList(e,t,i){return e.filter(e=>e.accounts.length>0).map((e,a)=>{let r=V.a.getConnectorById(e.connectorId),n=c.$.getConnectorImage(r)??"",s=eY.getAuthData(e);return e.accounts.map((r,l)=>{let c=this.isAccountLoading(e.connectorId,r.address);return(0,o.qy)`
            <wui-flex flexDirection="column">
              ${0!==a||0!==l?(0,o.qy)`<wui-separator></wui-separator>`:null}
              <wui-inactive-profile-wallet-item
                address=${r.address}
                alt=${e.connectorId}
                buttonLabel=${t?"Connect":"Switch"}
                buttonVariant=${t?"neutral":"accent"}
                rightIcon=${t?"bin":"off"}
                rightIconSize="sm"
                class=${t?"recent-connection":"active-connection"}
                data-testid=${t?"recent-connection":"active-connection"}
                imageSrc=${n}
                .iconBadge=${this.isSmartAccount(r.address)?e2:void 0}
                .iconBadgeSize=${this.isSmartAccount(r.address)?e1:void 0}
                .icon=${s.icon}
                .iconSize=${s.iconSize}
                .loading=${c}
                .showBalance=${!1}
                .charsStart=${e0}
                .charsEnd=${e3}
                @buttonClick=${()=>this.handleSwitchWallet(e,r.address,i)}
                @iconClick=${()=>this.handleWalletAction({connection:e,address:r.address,isRecentConnection:t,namespace:i})}
              ></wui-inactive-profile-wallet-item>
            </wui-flex>
          `})})}renderAddConnectionButton(e){if(!this.isMultiWalletEnabled()&&this.caipAddress||!this.hasAnyConnections(e))return null;let{title:t}=this.getChainLabelInfo(e);return(0,o.qy)`
      <wui-list-item
        variant="icon"
        iconVariant="overlay"
        icon="plus"
        iconSize="sm"
        ?chevron=${!0}
        @click=${()=>this.handleAddConnection(e)}
        data-testid="add-connection-button"
      >
        <wui-text variant="paragraph-500" color="fg-200">${t}</wui-text>
      </wui-list-item>
    `}renderEmptyState(e){let{title:t,description:i}=this.getChainLabelInfo(e);return(0,o.qy)`
      <wui-flex alignItems="flex-start" class="empty-template" data-testid="empty-template">
        <wui-flex
          flexDirection="column"
          alignItems="center"
          justifyContent="center"
          rowGap="s"
          class="empty-box"
        >
          <wui-icon-box
            size="lg"
            icon="wallet"
            background="gray"
            iconColor="fg-200"
            backgroundColor="glass-002"
          ></wui-icon-box>

          <wui-flex flexDirection="column" alignItems="center" justifyContent="center" gap="3xs">
            <wui-text color="fg-100" variant="paragraph-500" data-testid="empty-state-text"
              >No wallet connected</wui-text
            >
            <wui-text color="fg-200" variant="tiny-500" data-testid="empty-state-description"
              >${i}</wui-text
            >
          </wui-flex>

          <wui-button
            variant="neutral"
            size="md"
            @click=${()=>this.handleAddConnection(e)}
            data-testid="empty-state-button"
          >
            <wui-icon color="inherit" slot="iconLeft" name="plus"></wui-icon>
            ${t}
          </wui-button>
        </wui-flex>
      </wui-flex>
    `}handleTabChange(e){let t=this.namespaces[e];t&&(this.chainListener?.(),this.currentTab=this.namespaces.indexOf(t),this.namespace=t,this.caipAddress=s.W.getAccountData(t)?.caipAddress,this.profileName=s.W.getAccountData(t)?.profileName,this.chainListener=s.W.subscribeChainProp("accountState",e=>{this.caipAddress=e?.caipAddress},t))}async handleSwitchWallet(e,t,i){try{this.isSwitching=!0,this.lastSelectedConnectorId=e.connectorId,this.lastSelectedAddress=t,await X.x.switchConnection({connection:e,address:t,namespace:i,closeModalOnConnect:!1,onChange({hasSwitchedAccount:e,hasSwitchedWallet:t}){t?J.P.showSuccess("Wallet switched"):e&&J.P.showSuccess("Account switched")}})}catch(e){J.P.showError("Failed to switch wallet")}finally{this.isSwitching=!1}}handleWalletAction(e){let{connection:t,address:i,isRecentConnection:o,namespace:a}=e;o?(ea.i.deleteAddressFromConnection({connectorId:t.connectorId,address:i,namespace:a}),X.x.syncStorageConnections(),J.P.showSuccess("Wallet deleted")):this.handleDisconnect(a,{id:t.connectorId})}async handleDisconnect(e,{id:t}){try{await X.x.disconnect({id:t,namespace:e}),J.P.showSuccess("Wallet disconnected")}catch{J.P.showError("Failed to disconnect wallet")}}handleCopyAddress(e){u.w.copyToClopboard(e),J.P.showSuccess("Address copied")}handleMore(){Y.I.push("AccountSettings")}handleExternalLink(e){let t=this.caipNetwork?.blockExplorers?.default.url;t&&u.w.openHref(`${t}/address/${e}`,"_blank")}handleAddConnection(e){V.a.setFilterByNamespace(e),Y.I.push("Connect")}getChainLabelInfo(e){return e8[e]??{title:"Add Wallet",description:"Add your first wallet"}}isSmartAccount(e){if(!this.namespace)return!1;let t=this.user?.accounts?.find(e=>"smartAccount"===e.type);return!!t&&!!e&&eG.y.isLowerCaseMatch(t.address,e)}getPlainAddress(){return this.caipAddress?u.w.getPlainAddress(this.caipAddress):void 0}getActiveConnections(e){let t=this.activeConnectorIds[e],{connections:i}=eZ.b.getConnectionsData(e),[o]=i.filter(e=>eG.y.isLowerCaseMatch(e.connectorId,t));if(!t)return i;let a=e===_.o.CHAIN.BITCOIN,{address:r}=this.caipAddress?eF.C.parseCaipAddress(this.caipAddress):{},n=[...r?[r]:[]];return a&&o&&(n=o.accounts.map(e=>e.address)||[]),eZ.b.excludeConnectorAddressFromConnections({connectorId:t,addresses:n,connections:i})}hasAnyConnections(e){let t=this.getActiveConnections(e),{recentConnections:i}=eZ.b.getConnectionsData(e);return!!this.caipAddress||t.length>0||i.length>0}isAccountLoading(e,t){return eG.y.isLowerCaseMatch(this.lastSelectedConnectorId,e)&&eG.y.isLowerCaseMatch(this.lastSelectedAddress,t)&&this.isSwitching}getProfileContent(e){let{address:t,connections:i,connectorId:o,namespace:a}=e,[r]=i.filter(e=>eG.y.isLowerCaseMatch(e.connectorId,o));if(a===_.o.CHAIN.BITCOIN&&r?.accounts.every(e=>"string"==typeof e.type))return this.getBitcoinProfileContent(r.accounts,t);let n=eY.getAuthData({connectorId:o,accounts:[]});return[{address:t,tagLabel:"Active",tagVariant:"success",enableButton:!0,profileName:this.profileName,buttonType:"disconnect",buttonLabel:"Disconnect",buttonVariant:"neutral",...n.isAuth?{description:this.isSmartAccount(t)?"Smart Account":"EOA Account"}:{}}]}getBitcoinProfileContent(e,t){let i=e.length>1,o=this.getPlainAddress();return e.map(e=>{let a=eG.y.isLowerCaseMatch(e.address,o),r="PAYMENT";return"ordinal"===e.type&&(r="ORDINALS"),{address:e.address,tagLabel:eG.y.isLowerCaseMatch(e.address,t)?"Active":void 0,tagVariant:eG.y.isLowerCaseMatch(e.address,t)?"success":void 0,enableButton:!0,...i?{label:r,alignItems:"flex-end",buttonType:a?"disconnect":"switch",buttonLabel:a?"Disconnect":"Switch",buttonVariant:a?"neutral":"accent"}:{alignItems:"center",buttonType:"disconnect",buttonLabel:"Disconnect",buttonVariant:"neutral"}}})}removeScrollListener(){let e=this.shadowRoot?.querySelector(".wallet-list");e&&e.removeEventListener("scroll",()=>this.handleConnectListScroll())}handleConnectListScroll(){let e=this.shadowRoot?.querySelector(".wallet-list");e&&this.updateScrollOpacity(e)}isMultiWalletEnabled(){return!!this.remoteFeatures?.multiWallet}updateScrollOpacity(e){e.style.setProperty("--connect-scroll--top-opacity",p.z8.interpolate([0,50],e5,e.scrollTop).toString()),e.style.setProperty("--connect-scroll--bottom-opacity",p.z8.interpolate([0,50],e5,e.scrollHeight-e.scrollTop-e.offsetHeight).toString())}onConnectionsChange(){if(this.isMultiWalletEnabled()&&this.namespace){let{connections:e}=eZ.b.getConnectionsData(this.namespace);0===e.length&&Y.I.reset("ProfileWallets")}this.requestUpdate()}};e7.styles=eX,eQ([(0,a.wk)()],e7.prototype,"currentTab",void 0),eQ([(0,a.wk)()],e7.prototype,"namespace",void 0),eQ([(0,a.wk)()],e7.prototype,"namespaces",void 0),eQ([(0,a.wk)()],e7.prototype,"caipAddress",void 0),eQ([(0,a.wk)()],e7.prototype,"profileName",void 0),eQ([(0,a.wk)()],e7.prototype,"activeConnectorIds",void 0),eQ([(0,a.wk)()],e7.prototype,"lastSelectedAddress",void 0),eQ([(0,a.wk)()],e7.prototype,"lastSelectedConnectorId",void 0),eQ([(0,a.wk)()],e7.prototype,"isSwitching",void 0),eQ([(0,a.wk)()],e7.prototype,"caipNetwork",void 0),eQ([(0,a.wk)()],e7.prototype,"user",void 0),eQ([(0,a.wk)()],e7.prototype,"remoteFeatures",void 0),eQ([(0,a.wk)()],e7.prototype,"tabWidth",void 0),e7=eQ([(0,p.EM)("w3m-profile-wallets-view")],e7);var e9=i(22096);let te=(0,o.AH)`
  :host {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  label {
    position: relative;
    display: inline-block;
    width: 32px;
    height: 22px;
  }

  input {
    width: 0;
    height: 0;
    opacity: 0;
  }

  span {
    position: absolute;
    cursor: pointer;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: var(--wui-color-blue-100);
    border-width: 1px;
    border-style: solid;
    border-color: var(--wui-color-gray-glass-002);
    border-radius: 999px;
    transition:
      background-color var(--wui-ease-inout-power-1) var(--wui-duration-md),
      border-color var(--wui-ease-inout-power-1) var(--wui-duration-md);
    will-change: background-color, border-color;
  }

  span:before {
    position: absolute;
    content: '';
    height: 16px;
    width: 16px;
    left: 3px;
    top: 2px;
    background-color: var(--wui-color-inverse-100);
    transition: transform var(--wui-ease-inout-power-1) var(--wui-duration-lg);
    will-change: transform;
    border-radius: 50%;
  }

  input:checked + span {
    border-color: var(--wui-color-gray-glass-005);
    background-color: var(--wui-color-blue-100);
  }

  input:not(:checked) + span {
    background-color: var(--wui-color-gray-glass-010);
  }

  input:checked + span:before {
    transform: translateX(calc(100% - 7px));
  }
`;var tt=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ti=class extends o.WF{constructor(){super(...arguments),this.inputElementRef=(0,e9._)(),this.checked=void 0}render(){return(0,o.qy)`
      <label>
        <input
          ${(0,e9.K)(this.inputElementRef)}
          type="checkbox"
          ?checked=${(0,r.J)(this.checked)}
          @change=${this.dispatchChangeEvent.bind(this)}
        />
        <span></span>
      </label>
    `}dispatchChangeEvent(){this.dispatchEvent(new CustomEvent("switchChange",{detail:this.inputElementRef.value?.checked,bubbles:!0,composed:!0}))}};ti.styles=[w.W5,w.fD,w.ck,te],tt([(0,a.MZ)({type:Boolean})],ti.prototype,"checked",void 0),ti=tt([(0,b.E)("wui-switch")],ti);let to=(0,o.AH)`
  :host {
    height: 100%;
  }

  button {
    display: flex;
    align-items: center;
    justify-content: center;
    column-gap: var(--wui-spacing-1xs);
    padding: var(--wui-spacing-xs) var(--wui-spacing-s);
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xs);
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-002);
    transition: background-color var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: background-color;
    cursor: pointer;
  }

  wui-switch {
    pointer-events: none;
  }
`;var ta=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tr=class extends o.WF{constructor(){super(...arguments),this.checked=void 0}render(){return(0,o.qy)`
      <button>
        <wui-icon size="xl" name="walletConnectBrown"></wui-icon>
        <wui-switch ?checked=${(0,r.J)(this.checked)}></wui-switch>
      </button>
    `}};tr.styles=[w.W5,w.fD,to],ta([(0,a.MZ)({type:Boolean})],tr.prototype,"checked",void 0),tr=ta([(0,b.E)("wui-certified-switch")],tr);let tn=(0,o.AH)`
  button {
    background-color: var(--wui-color-fg-300);
    border-radius: var(--wui-border-radius-4xs);
    width: 16px;
    height: 16px;
  }

  button:disabled {
    background-color: var(--wui-color-bg-300);
  }

  wui-icon {
    color: var(--wui-color-bg-200) !important;
  }

  button:focus-visible {
    background-color: var(--wui-color-fg-250);
    border: 1px solid var(--wui-color-accent-100);
  }

  @media (hover: hover) and (pointer: fine) {
    button:hover:enabled {
      background-color: var(--wui-color-fg-250);
    }

    button:active:enabled {
      background-color: var(--wui-color-fg-225);
    }
  }
`;var ts=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tl=class extends o.WF{constructor(){super(...arguments),this.icon="copy"}render(){return(0,o.qy)`
      <button>
        <wui-icon color="inherit" size="xxs" name=${this.icon}></wui-icon>
      </button>
    `}};tl.styles=[w.W5,w.fD,tn],ts([(0,a.MZ)()],tl.prototype,"icon",void 0),tl=ts([(0,b.E)("wui-input-element")],tl),i(19493);let tc=(0,o.AH)`
  :host {
    position: relative;
    display: inline-block;
    width: 100%;
  }
`,td=class extends o.WF{constructor(){super(...arguments),this.inputComponentRef=(0,e9._)()}render(){return(0,o.qy)`
      <wui-input-text
        ${(0,e9.K)(this.inputComponentRef)}
        placeholder="Search wallet"
        icon="search"
        type="search"
        enterKeyHint="search"
        size="sm"
      >
        <wui-input-element @click=${this.clearValue} icon="close"></wui-input-element>
      </wui-input-text>
    `}clearValue(){let e=this.inputComponentRef.value,t=e?.inputElementRef.value;t&&(t.value="",t.focus(),t.dispatchEvent(new Event("input")))}};td.styles=[w.W5,tc],td=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,b.E)("wui-search-bar")],td);var tu=i(92929),th=i(31320);i(63270);let tp=(0,o.AH)`
  :host {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 104px;
    row-gap: var(--wui-spacing-xs);
    padding: var(--wui-spacing-xs) 10px;
    background-color: var(--wui-color-gray-glass-002);
    border-radius: clamp(0px, var(--wui-border-radius-xs), 20px);
    position: relative;
  }

  wui-shimmer[data-type='network'] {
    border: none;
    -webkit-clip-path: var(--wui-path-network);
    clip-path: var(--wui-path-network);
  }

  svg {
    position: absolute;
    width: 48px;
    height: 54px;
    z-index: 1;
  }

  svg > path {
    stroke: var(--wui-color-gray-glass-010);
    stroke-width: 1px;
  }

  @media (max-width: 350px) {
    :host {
      width: 100%;
    }
  }
`;var tw=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tg=class extends o.WF{constructor(){super(...arguments),this.type="wallet"}render(){return(0,o.qy)`
      ${this.shimmerTemplate()}
      <wui-shimmer width="56px" height="20px" borderRadius="xs"></wui-shimmer>
    `}shimmerTemplate(){return"network"===this.type?(0,o.qy)` <wui-shimmer
          data-type=${this.type}
          width="48px"
          height="54px"
          borderRadius="xs"
        ></wui-shimmer>
        ${th.a}`:(0,o.qy)`<wui-shimmer width="56px" height="56px" borderRadius="xs"></wui-shimmer>`}};tg.styles=[w.W5,w.fD,tp],tw([(0,a.MZ)()],tg.prototype,"type",void 0),tg=tw([(0,b.E)("wui-card-select-loader")],tg);let tb=(0,o.AH)`
  :host {
    display: grid;
    width: inherit;
    height: inherit;
  }
`;var tf=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tm=class extends o.WF{render(){return this.style.cssText=`
      grid-template-rows: ${this.gridTemplateRows};
      grid-template-columns: ${this.gridTemplateColumns};
      justify-items: ${this.justifyItems};
      align-items: ${this.alignItems};
      justify-content: ${this.justifyContent};
      align-content: ${this.alignContent};
      column-gap: ${this.columnGap&&`var(--wui-spacing-${this.columnGap})`};
      row-gap: ${this.rowGap&&`var(--wui-spacing-${this.rowGap})`};
      gap: ${this.gap&&`var(--wui-spacing-${this.gap})`};
      padding-top: ${this.padding&&g.Z.getSpacingStyles(this.padding,0)};
      padding-right: ${this.padding&&g.Z.getSpacingStyles(this.padding,1)};
      padding-bottom: ${this.padding&&g.Z.getSpacingStyles(this.padding,2)};
      padding-left: ${this.padding&&g.Z.getSpacingStyles(this.padding,3)};
      margin-top: ${this.margin&&g.Z.getSpacingStyles(this.margin,0)};
      margin-right: ${this.margin&&g.Z.getSpacingStyles(this.margin,1)};
      margin-bottom: ${this.margin&&g.Z.getSpacingStyles(this.margin,2)};
      margin-left: ${this.margin&&g.Z.getSpacingStyles(this.margin,3)};
    `,(0,o.qy)`<slot></slot>`}};tm.styles=[w.W5,tb],tf([(0,a.MZ)()],tm.prototype,"gridTemplateRows",void 0),tf([(0,a.MZ)()],tm.prototype,"gridTemplateColumns",void 0),tf([(0,a.MZ)()],tm.prototype,"justifyItems",void 0),tf([(0,a.MZ)()],tm.prototype,"alignItems",void 0),tf([(0,a.MZ)()],tm.prototype,"justifyContent",void 0),tf([(0,a.MZ)()],tm.prototype,"alignContent",void 0),tf([(0,a.MZ)()],tm.prototype,"columnGap",void 0),tf([(0,a.MZ)()],tm.prototype,"rowGap",void 0),tf([(0,a.MZ)()],tm.prototype,"gap",void 0),tf([(0,a.MZ)()],tm.prototype,"padding",void 0),tf([(0,a.MZ)()],tm.prototype,"margin",void 0),tm=tf([(0,b.E)("wui-grid")],tm);var tv=i(54028);i(26400),i(18128);let ty=(0,o.AH)`
  button {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    cursor: pointer;
    width: 104px;
    row-gap: var(--wui-spacing-xs);
    padding: var(--wui-spacing-s) var(--wui-spacing-0);
    background-color: var(--wui-color-gray-glass-002);
    border-radius: clamp(0px, var(--wui-border-radius-xs), 20px);
    transition:
      color var(--wui-duration-lg) var(--wui-ease-out-power-1),
      background-color var(--wui-duration-lg) var(--wui-ease-out-power-1),
      border-radius var(--wui-duration-lg) var(--wui-ease-out-power-1);
    will-change: background-color, color, border-radius;
    outline: none;
    border: none;
  }

  button > wui-flex > wui-text {
    color: var(--wui-color-fg-100);
    max-width: 86px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    justify-content: center;
  }

  button > wui-flex > wui-text.certified {
    max-width: 66px;
  }

  button:hover:enabled {
    background-color: var(--wui-color-gray-glass-005);
  }

  button:disabled > wui-flex > wui-text {
    color: var(--wui-color-gray-glass-015);
  }

  [data-selected='true'] {
    background-color: var(--wui-color-accent-glass-020);
  }

  @media (hover: hover) and (pointer: fine) {
    [data-selected='true']:hover:enabled {
      background-color: var(--wui-color-accent-glass-015);
    }
  }

  [data-selected='true']:active:enabled {
    background-color: var(--wui-color-accent-glass-010);
  }

  @media (max-width: 350px) {
    button {
      width: 100%;
    }
  }
`;var tx=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tk=class extends o.WF{constructor(){super(),this.observer=new IntersectionObserver(()=>void 0),this.visible=!1,this.imageSrc=void 0,this.imageLoading=!1,this.wallet=void 0,this.observer=new IntersectionObserver(e=>{e.forEach(e=>{e.isIntersecting?(this.visible=!0,this.fetchImageSrc()):this.visible=!1})},{threshold:.01})}firstUpdated(){this.observer.observe(this)}disconnectedCallback(){this.observer.disconnect()}render(){let e=this.wallet?.badge_type==="certified";return(0,o.qy)`
      <button>
        ${this.imageTemplate()}
        <wui-flex flexDirection="row" alignItems="center" justifyContent="center" gap="3xs">
          <wui-text
            variant="tiny-500"
            color="inherit"
            class=${(0,r.J)(e?"certified":void 0)}
            >${this.wallet?.name}</wui-text
          >
          ${e?(0,o.qy)`<wui-icon size="sm" name="walletConnectBrown"></wui-icon>`:null}
        </wui-flex>
      </button>
    `}imageTemplate(){return(this.visible||this.imageSrc)&&!this.imageLoading?(0,o.qy)`
      <wui-wallet-image
        size="md"
        imageSrc=${(0,r.J)(this.imageSrc)}
        name=${this.wallet?.name}
        .installed=${this.wallet?.installed}
        badgeSize="sm"
      >
      </wui-wallet-image>
    `:this.shimmerTemplate()}shimmerTemplate(){return(0,o.qy)`<wui-shimmer width="56px" height="56px" borderRadius="xs"></wui-shimmer>`}async fetchImageSrc(){!this.wallet||(this.imageSrc=c.$.getWalletImage(this.wallet),this.imageSrc||(this.imageLoading=!0,this.imageSrc=await c.$.fetchWalletImage(this.wallet.image_id),this.imageLoading=!1))}};tk.styles=ty,tx([(0,a.wk)()],tk.prototype,"visible",void 0),tx([(0,a.wk)()],tk.prototype,"imageSrc",void 0),tx([(0,a.wk)()],tk.prototype,"imageLoading",void 0),tx([(0,a.MZ)()],tk.prototype,"wallet",void 0),tk=tx([(0,p.EM)("w3m-all-wallets-list-item")],tk);let tC=(0,o.AH)`
  wui-grid {
    max-height: clamp(360px, 400px, 80vh);
    overflow: scroll;
    scrollbar-width: none;
    grid-auto-rows: min-content;
    grid-template-columns: repeat(auto-fill, 104px);
  }

  @media (max-width: 350px) {
    wui-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  wui-grid[data-scroll='false'] {
    overflow: hidden;
  }

  wui-grid::-webkit-scrollbar {
    display: none;
  }

  wui-loading-spinner {
    padding-top: var(--wui-spacing-l);
    padding-bottom: var(--wui-spacing-l);
    justify-content: center;
    grid-column: 1 / span 4;
  }
`;var t$=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tI="local-paginator",tE=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.paginationObserver=void 0,this.loading=!tu.N.state.wallets.length,this.wallets=tu.N.state.wallets,this.recommended=tu.N.state.recommended,this.featured=tu.N.state.featured,this.filteredWallets=tu.N.state.filteredWallets,this.unsubscribe.push(tu.N.subscribeKey("wallets",e=>this.wallets=e),tu.N.subscribeKey("recommended",e=>this.recommended=e),tu.N.subscribeKey("featured",e=>this.featured=e),tu.N.subscribeKey("filteredWallets",e=>this.filteredWallets=e))}firstUpdated(){this.initialFetch(),this.createPaginationObserver()}disconnectedCallback(){this.unsubscribe.forEach(e=>e()),this.paginationObserver?.disconnect()}render(){return(0,o.qy)`
      <wui-grid
        data-scroll=${!this.loading}
        .padding=${["0","s","s","s"]}
        columnGap="xxs"
        rowGap="l"
        justifyContent="space-between"
      >
        ${this.loading?this.shimmerTemplate(16):this.walletsTemplate()}
        ${this.paginationLoaderTemplate()}
      </wui-grid>
    `}async initialFetch(){this.loading=!0;let e=this.shadowRoot?.querySelector("wui-grid");e&&(await tu.N.fetchWalletsByPage({page:1}),await e.animate([{opacity:1},{opacity:0}],{duration:200,fill:"forwards",easing:"ease"}).finished,this.loading=!1,e.animate([{opacity:0},{opacity:1}],{duration:200,fill:"forwards",easing:"ease"}))}shimmerTemplate(e,t){return[...Array(e)].map(()=>(0,o.qy)`
        <wui-card-select-loader type="wallet" id=${(0,r.J)(t)}></wui-card-select-loader>
      `)}getWallets(){let e=[...this.featured,...this.recommended];this.filteredWallets?.length>0?e.push(...this.filteredWallets):e.push(...this.wallets);let t=u.w.uniqueBy(e,"id"),i=tv.A.markWalletsAsInstalled(t);return tv.A.markWalletsWithDisplayIndex(i)}walletsTemplate(){return this.getWallets().map(e=>(0,o.qy)`
        <w3m-all-wallets-list-item
          @click=${()=>this.onConnectWallet(e)}
          .wallet=${e}
        ></w3m-all-wallets-list-item>
      `)}paginationLoaderTemplate(){let{wallets:e,recommended:t,featured:i,count:o}=tu.N.state,a=window.innerWidth<352?3:4,r=e.length+t.length,n=Math.ceil(r/a)*a-r+a;return(n-=e.length?i.length%a:0,0===o&&i.length>0)?null:0===o||[...i,...e,...t].length<o?this.shimmerTemplate(n,tI):null}createPaginationObserver(){let e=this.shadowRoot?.querySelector(`#${tI}`);e&&(this.paginationObserver=new IntersectionObserver(([e])=>{if(e?.isIntersecting&&!this.loading){let{page:e,count:t,wallets:i}=tu.N.state;i.length<t&&tu.N.fetchWalletsByPage({page:e+1})}}),this.paginationObserver.observe(e))}onConnectWallet(e){V.a.selectWalletConnector(e)}};tE.styles=tC,t$([(0,a.wk)()],tE.prototype,"loading",void 0),t$([(0,a.wk)()],tE.prototype,"wallets",void 0),t$([(0,a.wk)()],tE.prototype,"recommended",void 0),t$([(0,a.wk)()],tE.prototype,"featured",void 0),t$([(0,a.wk)()],tE.prototype,"filteredWallets",void 0),tE=t$([(0,p.EM)("w3m-all-wallets-list")],tE),i(88844);let tS=(0,o.AH)`
  wui-grid,
  wui-loading-spinner,
  wui-flex {
    height: 360px;
  }

  wui-grid {
    overflow: scroll;
    scrollbar-width: none;
    grid-auto-rows: min-content;
    grid-template-columns: repeat(auto-fill, 104px);
  }

  wui-grid[data-scroll='false'] {
    overflow: hidden;
  }

  wui-grid::-webkit-scrollbar {
    display: none;
  }

  wui-loading-spinner {
    justify-content: center;
    align-items: center;
  }

  @media (max-width: 350px) {
    wui-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
`;var tA=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tW=class extends o.WF{constructor(){super(...arguments),this.prevQuery="",this.prevBadge=void 0,this.loading=!0,this.query=""}render(){return this.onSearch(),this.loading?(0,o.qy)`<wui-loading-spinner color="accent-100"></wui-loading-spinner>`:this.walletsTemplate()}async onSearch(){(this.query.trim()!==this.prevQuery.trim()||this.badge!==this.prevBadge)&&(this.prevQuery=this.query,this.prevBadge=this.badge,this.loading=!0,await tu.N.searchWallet({search:this.query,badge:this.badge}),this.loading=!1)}walletsTemplate(){let{search:e}=tu.N.state,t=tv.A.markWalletsAsInstalled(e);return e.length?(0,o.qy)`
      <wui-grid
        data-testid="wallet-list"
        .padding=${["0","s","s","s"]}
        rowGap="l"
        columnGap="xs"
        justifyContent="space-between"
      >
        ${t.map(e=>(0,o.qy)`
            <w3m-all-wallets-list-item
              @click=${()=>this.onConnectWallet(e)}
              .wallet=${e}
              data-testid="wallet-search-item-${e.id}"
            ></w3m-all-wallets-list-item>
          `)}
      </wui-grid>
    `:(0,o.qy)`
        <wui-flex
          data-testid="no-wallet-found"
          justifyContent="center"
          alignItems="center"
          gap="s"
          flexDirection="column"
        >
          <wui-icon-box
            size="lg"
            iconColor="fg-200"
            backgroundColor="fg-300"
            icon="wallet"
            background="transparent"
          ></wui-icon-box>
          <wui-text data-testid="no-wallet-found-text" color="fg-200" variant="paragraph-500">
            No Wallet found
          </wui-text>
        </wui-flex>
      `}onConnectWallet(e){V.a.selectWalletConnector(e)}};tW.styles=tS,tA([(0,a.wk)()],tW.prototype,"loading",void 0),tA([(0,a.MZ)()],tW.prototype,"query",void 0),tA([(0,a.MZ)()],tW.prototype,"badge",void 0),tW=tA([(0,p.EM)("w3m-all-wallets-search")],tW);var tT=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tR=class extends o.WF{constructor(){super(...arguments),this.search="",this.onDebouncedSearch=u.w.debounce(e=>{this.search=e})}render(){let e=this.search.length>=2;return(0,o.qy)`
      <wui-flex .padding=${["0","s","s","s"]} gap="xs">
        <wui-search-bar @inputChange=${this.onInputChange.bind(this)}></wui-search-bar>
        <wui-certified-switch
          ?checked=${this.badge}
          @click=${this.onClick.bind(this)}
          data-testid="wui-certified-switch"
        ></wui-certified-switch>
        ${this.qrButtonTemplate()}
      </wui-flex>
      ${e||this.badge?(0,o.qy)`<w3m-all-wallets-search
            query=${this.search}
            badge=${(0,r.J)(this.badge)}
          ></w3m-all-wallets-search>`:(0,o.qy)`<w3m-all-wallets-list badge=${(0,r.J)(this.badge)}></w3m-all-wallets-list>`}
    `}onInputChange(e){this.onDebouncedSearch(e.detail)}onClick(){if("certified"===this.badge){this.badge=void 0;return}this.badge="certified",J.P.showSvg("Only WalletConnect certified",{icon:"walletConnectBrown",iconColor:"accent-100"})}qrButtonTemplate(){return u.w.isMobile()?(0,o.qy)`
        <wui-icon-box
          size="lg"
          iconSize="xl"
          iconColor="accent-100"
          backgroundColor="accent-100"
          icon="qrCode"
          background="transparent"
          border
          borderColor="wui-accent-glass-010"
          @click=${this.onWalletConnectQr.bind(this)}
        ></wui-icon-box>
      `:null}onWalletConnectQr(){Y.I.push("ConnectingWalletConnect")}};tT([(0,a.wk)()],tR.prototype,"search",void 0),tT([(0,a.wk)()],tR.prototype,"badge",void 0),tR=tT([(0,p.EM)("w3m-all-wallets-view")],tR);var tO=i(97734),tN=i(74016);let tM=(0,o.AH)`
  button {
    column-gap: var(--wui-spacing-s);
    padding: 16.5px var(--wui-spacing-l) 16.5px var(--wui-spacing-xs);
    width: 100%;
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xs);
    color: var(--wui-color-fg-100);
    justify-content: center;
    align-items: center;
  }

  button:disabled {
    background-color: var(--wui-color-gray-glass-015);
    color: var(--wui-color-gray-glass-015);
  }
`;var tP=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tj=class extends o.WF{constructor(){super(...arguments),this.text="",this.disabled=!1,this.tabIdx=void 0}render(){return(0,o.qy)`
      <button ?disabled=${this.disabled} tabindex=${(0,r.J)(this.tabIdx)}>
        <wui-text align="center" variant="paragraph-500" color="inherit">${this.text}</wui-text>
      </button>
    `}};tj.styles=[w.W5,w.fD,tM],tP([(0,a.MZ)()],tj.prototype,"text",void 0),tP([(0,a.MZ)({type:Boolean})],tj.prototype,"disabled",void 0),tP([(0,a.MZ)()],tj.prototype,"tabIdx",void 0),tj=tP([(0,b.E)("wui-list-button")],tj);var tq=i(13226);i(48657);let tD=(0,o.AH)`
  wui-separator {
    margin: var(--wui-spacing-s) calc(var(--wui-spacing-s) * -1);
    width: calc(100% + var(--wui-spacing-s) * 2);
  }

  wui-email-input {
    width: 100%;
  }

  form {
    width: 100%;
    display: block;
    position: relative;
  }

  wui-icon-link,
  wui-loading-spinner {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
  }

  wui-icon-link {
    right: var(--wui-spacing-xs);
  }

  wui-loading-spinner {
    right: var(--wui-spacing-m);
  }

  wui-text {
    margin: var(--wui-spacing-xxs) var(--wui-spacing-m) var(--wui-spacing-0) var(--wui-spacing-m);
  }
`;var tL=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tU=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.formRef=(0,e9._)(),this.email="",this.loading=!1,this.error="",this.remoteFeatures=n.H.state.remoteFeatures,this.unsubscribe.push(n.H.subscribeKey("remoteFeatures",e=>{this.remoteFeatures=e}))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}firstUpdated(){this.formRef.value?.addEventListener("keydown",e=>{"Enter"===e.key&&this.onSubmitEmail(e)})}render(){let e=X.x.hasAnyConnection(_.o.CONNECTOR_ID.AUTH);return(0,o.qy)`
      <form ${(0,e9.K)(this.formRef)} @submit=${this.onSubmitEmail.bind(this)}>
        <wui-email-input
          @focus=${this.onFocusEvent.bind(this)}
          .disabled=${this.loading}
          @inputChange=${this.onEmailInputChange.bind(this)}
          tabIdx=${(0,r.J)(this.tabIdx)}
          ?disabled=${e}
        >
        </wui-email-input>

        ${this.submitButtonTemplate()}${this.loadingTemplate()}
        <input type="submit" hidden />
      </form>
      ${this.templateError()}
    `}submitButtonTemplate(){return!this.loading&&this.email.length>3?(0,o.qy)`
          <wui-icon-link
            size="sm"
            icon="chevronRight"
            iconcolor="accent-100"
            @click=${this.onSubmitEmail.bind(this)}
          >
          </wui-icon-link>
        `:null}loadingTemplate(){return this.loading?(0,o.qy)`<wui-loading-spinner size="md" color="accent-100"></wui-loading-spinner>`:null}templateError(){return this.error?(0,o.qy)`<wui-text variant="tiny-500" color="error-100">${this.error}</wui-text>`:null}onEmailInputChange(e){this.email=e.detail.trim(),this.error=""}async onSubmitEmail(e){if(!_.o.AUTH_CONNECTOR_SUPPORTED_CHAINS.find(e=>e===s.W.state.activeChain)){let e=s.W.getFirstCaipNetworkSupportsAuthConnector();if(e)return void Y.I.push("SwitchNetwork",{network:e})}try{if(this.loading)return;this.loading=!0,e.preventDefault();let t=V.a.getAuthConnector();if(!t)throw Error("w3m-email-login-widget: Auth connector not found");let{action:i}=await t.provider.connectEmail({email:this.email});if(j.E.sendEvent({type:"track",event:"EMAIL_SUBMITTED"}),"VERIFY_OTP"===i)j.E.sendEvent({type:"track",event:"EMAIL_VERIFICATION_CODE_SENT"}),Y.I.push("EmailVerifyOtp",{email:this.email});else if("VERIFY_DEVICE"===i)Y.I.push("EmailVerifyDevice",{email:this.email});else if("CONNECT"===i){let e=this.remoteFeatures?.multiWallet;await X.x.connectExternal(t,s.W.state.activeChain),e?(Y.I.replace("ProfileWallets"),J.P.showSuccess("New Wallet Added")):Y.I.replace("Account")}}catch(t){let e=u.w.parseError(t);e?.includes("Invalid email")?this.error="Invalid email. Try again.":J.P.showError(t)}finally{this.loading=!1}}onFocusEvent(){j.E.sendEvent({type:"track",event:"EMAIL_LOGIN_SELECTED"})}};tU.styles=tD,tL([(0,a.MZ)()],tU.prototype,"tabIdx",void 0),tL([(0,a.wk)()],tU.prototype,"email",void 0),tL([(0,a.wk)()],tU.prototype,"loading",void 0),tL([(0,a.wk)()],tU.prototype,"error",void 0),tL([(0,a.wk)()],tU.prototype,"remoteFeatures",void 0),tU=tL([(0,p.EM)("w3m-email-login-widget")],tU),i(73989),i(62791);var tz=i(25663),tF=i(89373);i(32297),i(86212);let tZ=(0,o.AH)`
  :host {
    display: block;
    width: 100%;
  }

  button {
    width: 100%;
    height: 56px;
    background: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xs);
  }
`;var tB=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tH=class extends o.WF{constructor(){super(...arguments),this.logo="google",this.disabled=!1,this.tabIdx=void 0}render(){return(0,o.qy)`
      <button ?disabled=${this.disabled} tabindex=${(0,r.J)(this.tabIdx)}>
        <wui-logo logo=${this.logo}></wui-logo>
      </button>
    `}};tH.styles=[w.W5,w.fD,tZ],tB([(0,a.MZ)()],tH.prototype,"logo",void 0),tB([(0,a.MZ)({type:Boolean})],tH.prototype,"disabled",void 0),tB([(0,a.MZ)()],tH.prototype,"tabIdx",void 0),tH=tB([(0,b.E)("wui-logo-select")],tH);var t_=i(64495);let tV=(0,o.AH)`
  wui-separator {
    margin: var(--wui-spacing-m) calc(var(--wui-spacing-m) * -1) var(--wui-spacing-m)
      calc(var(--wui-spacing-m) * -1);
    width: calc(100% + var(--wui-spacing-s) * 2);
  }
`;var tK=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tJ=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.walletGuide="get-started",this.tabIdx=void 0,this.connectors=V.a.state.connectors,this.remoteFeatures=n.H.state.remoteFeatures,this.authConnector=this.connectors.find(e=>"AUTH"===e.type),this.isPwaLoading=!1,this.unsubscribe.push(V.a.subscribeKey("connectors",e=>{this.connectors=e,this.authConnector=this.connectors.find(e=>"AUTH"===e.type)}),n.H.subscribeKey("remoteFeatures",e=>this.remoteFeatures=e))}connectedCallback(){super.connectedCallback(),this.handlePwaFrameLoad()}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){return(0,o.qy)`
      <wui-flex
        class="container"
        flexDirection="column"
        gap="xs"
        data-testid="w3m-social-login-widget"
      >
        ${this.topViewTemplate()}${this.bottomViewTemplate()}
      </wui-flex>
    `}topViewTemplate(){let e="explore"===this.walletGuide,t=this.remoteFeatures?.socials;return!t&&e?(t=K.oU.DEFAULT_SOCIALS,this.renderTopViewContent(t)):t?this.renderTopViewContent(t):null}renderTopViewContent(e){return 2===e.length?(0,o.qy)` <wui-flex gap="xs">
        ${e.slice(0,2).map(e=>(0,o.qy)`<wui-logo-select
              data-testid=${`social-selector-${e}`}
              @click=${()=>{this.onSocialClick(e)}}
              logo=${e}
              tabIdx=${(0,r.J)(this.tabIdx)}
              ?disabled=${this.isPwaLoading||this.hasConnection()}
            ></wui-logo-select>`)}
      </wui-flex>`:(0,o.qy)` <wui-list-social
      data-testid=${`social-selector-${e[0]}`}
      @click=${()=>{this.onSocialClick(e[0])}}
      logo=${(0,r.J)(e[0])}
      align="center"
      name=${`Continue with ${e[0]}`}
      tabIdx=${(0,r.J)(this.tabIdx)}
      ?disabled=${this.isPwaLoading||this.hasConnection()}
    ></wui-list-social>`}bottomViewTemplate(){let e=this.remoteFeatures?.socials,t="explore"===this.walletGuide;return(this.authConnector&&e&&0!==e.length||!t||(e=K.oU.DEFAULT_SOCIALS),!e||e.length<=2)?null:e&&e.length>6?(0,o.qy)`<wui-flex gap="xs">
        ${e.slice(1,5).map(e=>(0,o.qy)`<wui-logo-select
              data-testid=${`social-selector-${e}`}
              @click=${()=>{this.onSocialClick(e)}}
              logo=${e}
              tabIdx=${(0,r.J)(this.tabIdx)}
              ?focusable=${void 0!==this.tabIdx&&this.tabIdx>=0}
              ?disabled=${this.isPwaLoading||this.hasConnection()}
            ></wui-logo-select>`)}
        <wui-logo-select
          logo="more"
          tabIdx=${(0,r.J)(this.tabIdx)}
          @click=${this.onMoreSocialsClick.bind(this)}
          ?disabled=${this.isPwaLoading||this.hasConnection()}
          data-testid="social-selector-more"
        ></wui-logo-select>
      </wui-flex>`:e?(0,o.qy)`<wui-flex gap="xs">
      ${e.slice(1,e.length).map(e=>(0,o.qy)`<wui-logo-select
            data-testid=${`social-selector-${e}`}
            @click=${()=>{this.onSocialClick(e)}}
            logo=${e}
            tabIdx=${(0,r.J)(this.tabIdx)}
            ?focusable=${void 0!==this.tabIdx&&this.tabIdx>=0}
            ?disabled=${this.isPwaLoading||this.hasConnection()}
          ></wui-logo-select>`)}
    </wui-flex>`:null}onMoreSocialsClick(){Y.I.push("ConnectSocials")}async onSocialClick(e){if(!_.o.AUTH_CONNECTOR_SUPPORTED_CHAINS.find(e=>e===s.W.state.activeChain)){let e=s.W.getFirstCaipNetworkSupportsAuthConnector();if(e)return void Y.I.push("SwitchNetwork",{network:e})}e&&await (0,tF.Up)(e)}async handlePwaFrameLoad(){if(u.w.isPWA()){this.isPwaLoading=!0;try{this.authConnector?.provider instanceof t_.Y&&await this.authConnector.provider.init()}catch(e){tz.h.open({displayMessage:"Error loading embedded wallet in PWA",debugMessage:e.message},"error")}finally{this.isPwaLoading=!1}}}hasConnection(){return X.x.hasAnyConnection(_.o.CONNECTOR_ID.AUTH)}};tJ.styles=tV,tK([(0,a.MZ)()],tJ.prototype,"walletGuide",void 0),tK([(0,a.MZ)()],tJ.prototype,"tabIdx",void 0),tK([(0,a.wk)()],tJ.prototype,"connectors",void 0),tK([(0,a.wk)()],tJ.prototype,"remoteFeatures",void 0),tK([(0,a.wk)()],tJ.prototype,"authConnector",void 0),tK([(0,a.wk)()],tJ.prototype,"isPwaLoading",void 0),tJ=tK([(0,p.EM)("w3m-social-login-widget")],tJ),i(29236);let tG=(0,o.AH)`
  wui-flex {
    width: 100%;
  }

  .wallet-guide {
    width: 100%;
  }

  .chip-box {
    width: fit-content;
    background-color: var(--wui-color-gray-glass-005);
    border-radius: var(--wui-border-radius-3xl);
  }
`;var tY=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let tX=class extends o.WF{constructor(){super(...arguments),this.walletGuide="get-started"}render(){return"explore"===this.walletGuide?(0,o.qy)`<wui-flex
          class="wallet-guide"
          flexDirection="column"
          alignItems="center"
          justifyContent="center"
          rowGap="xs"
          data-testid="w3m-wallet-guide-explore"
        >
          <wui-text variant="small-400" color="fg-200" align="center">
            Looking for a self-custody wallet?
          </wui-text>

          <wui-flex class="chip-box">
            <wui-chip
              imageIcon="walletConnectLightBrown"
              icon="externalLink"
              variant="transparent"
              href="https://walletguide.walletconnect.network"
              title="Find one on WalletGuide"
            ></wui-chip>
          </wui-flex>
        </wui-flex>`:(0,o.qy)`<wui-flex
          columnGap="4xs"
          flexDirection="row"
          alignItems="center"
          justifyContent="center"
          .padding=${["s","0","s","0"]}
        >
          <wui-text variant="small-400" class="title" color="fg-200"
            >Haven't got a wallet?</wui-text
          >
          <wui-link
            data-testid="w3m-wallet-guide-get-started"
            color="blue-100"
            class="get-started-link"
            @click=${this.onGetStarted}
            tabIdx=${(0,r.J)(this.tabIdx)}
          >
            Get started
          </wui-link>
        </wui-flex>`}onGetStarted(){Y.I.push("Create")}};tX.styles=tG,tY([(0,a.MZ)()],tX.prototype,"tabIdx",void 0),tY([(0,a.MZ)()],tX.prototype,"walletGuide",void 0),tX=tY([(0,p.EM)("w3m-wallet-guide")],tX);let tQ=(0,o.AH)`
  :host {
    position: relative;
    border-radius: var(--wui-border-radius-xxs);
    width: 40px;
    height: 40px;
    overflow: hidden;
    background: var(--wui-color-gray-glass-002);
    display: flex;
    justify-content: center;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--wui-spacing-4xs);
    padding: 3.75px !important;
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

  :host > wui-wallet-image {
    width: 14px;
    height: 14px;
    border-radius: var(--wui-border-radius-5xs);
  }

  :host > wui-flex {
    padding: 2px;
    position: fixed;
    overflow: hidden;
    left: 34px;
    bottom: 8px;
    background: var(--dark-background-150, #1e1f1f);
    border-radius: 50%;
    z-index: 2;
    display: flex;
  }
`;var t0=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let t3=class extends o.WF{constructor(){super(...arguments),this.walletImages=[]}render(){let e=this.walletImages.length<4;return(0,o.qy)`${this.walletImages.slice(0,4).map(({src:e,walletName:t})=>(0,o.qy)`
            <wui-wallet-image
              size="inherit"
              imageSrc=${e}
              name=${(0,r.J)(t)}
            ></wui-wallet-image>
          `)}
      ${e?[...Array(4-this.walletImages.length)].map(()=>(0,o.qy)` <wui-wallet-image size="inherit" name=""></wui-wallet-image>`):null}
      <wui-flex>
        <wui-icon-box
          size="xxs"
          iconSize="xxs"
          iconcolor="success-100"
          backgroundcolor="success-100"
          icon="checkmark"
          background="opaque"
        ></wui-icon-box>
      </wui-flex>`}};t3.styles=[w.W5,tQ],t0([(0,a.MZ)({type:Array})],t3.prototype,"walletImages",void 0),t3=t0([(0,b.E)("wui-all-wallets-image")],t3);let t1=(0,o.AH)`
  button {
    column-gap: var(--wui-spacing-s);
    padding: 7px var(--wui-spacing-l) 7px var(--wui-spacing-xs);
    width: 100%;
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xs);
    color: var(--wui-color-fg-100);
  }

  button > wui-text:nth-child(2) {
    display: flex;
    flex: 1;
  }

  button:disabled {
    background-color: var(--wui-color-gray-glass-015);
    color: var(--wui-color-gray-glass-015);
  }

  button:disabled > wui-tag {
    background-color: var(--wui-color-gray-glass-010);
    color: var(--wui-color-fg-300);
  }

  wui-icon {
    color: var(--wui-color-fg-200) !important;
  }
`;var t2=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let t5=class extends o.WF{constructor(){super(...arguments),this.walletImages=[],this.imageSrc="",this.name="",this.tabIdx=void 0,this.installed=!1,this.disabled=!1,this.showAllWallets=!1,this.loading=!1,this.loadingSpinnerColor="accent-100"}render(){return(0,o.qy)`
      <button ?disabled=${this.disabled} tabindex=${(0,r.J)(this.tabIdx)}>
        ${this.templateAllWallets()} ${this.templateWalletImage()}
        <wui-text variant="paragraph-500" color="inherit">${this.name}</wui-text>
        ${this.templateStatus()}
      </button>
    `}templateAllWallets(){return this.showAllWallets&&this.imageSrc?(0,o.qy)` <wui-all-wallets-image .imageeSrc=${this.imageSrc}> </wui-all-wallets-image> `:this.showAllWallets&&this.walletIcon?(0,o.qy)` <wui-wallet-image .walletIcon=${this.walletIcon} size="sm"> </wui-wallet-image> `:null}templateWalletImage(){return!this.showAllWallets&&this.imageSrc?(0,o.qy)`<wui-wallet-image
        size="sm"
        imageSrc=${this.imageSrc}
        name=${this.name}
        .installed=${this.installed}
      ></wui-wallet-image>`:this.showAllWallets||this.imageSrc?null:(0,o.qy)`<wui-wallet-image size="sm" name=${this.name}></wui-wallet-image>`}templateStatus(){return this.loading?(0,o.qy)`<wui-loading-spinner
        size="lg"
        color=${this.loadingSpinnerColor}
      ></wui-loading-spinner>`:this.tagLabel&&this.tagVariant?(0,o.qy)`<wui-tag variant=${this.tagVariant}>${this.tagLabel}</wui-tag>`:this.icon?(0,o.qy)`<wui-icon color="inherit" size="sm" name=${this.icon}></wui-icon>`:null}};t5.styles=[w.W5,w.fD,t1],t2([(0,a.MZ)({type:Array})],t5.prototype,"walletImages",void 0),t2([(0,a.MZ)()],t5.prototype,"imageSrc",void 0),t2([(0,a.MZ)()],t5.prototype,"name",void 0),t2([(0,a.MZ)()],t5.prototype,"tagLabel",void 0),t2([(0,a.MZ)()],t5.prototype,"tagVariant",void 0),t2([(0,a.MZ)()],t5.prototype,"icon",void 0),t2([(0,a.MZ)()],t5.prototype,"walletIcon",void 0),t2([(0,a.MZ)()],t5.prototype,"tabIdx",void 0),t2([(0,a.MZ)({type:Boolean})],t5.prototype,"installed",void 0),t2([(0,a.MZ)({type:Boolean})],t5.prototype,"disabled",void 0),t2([(0,a.MZ)({type:Boolean})],t5.prototype,"showAllWallets",void 0),t2([(0,a.MZ)({type:Boolean})],t5.prototype,"loading",void 0),t2([(0,a.MZ)({type:String})],t5.prototype,"loadingSpinnerColor",void 0),t5=t2([(0,b.E)("wui-list-wallet")],t5);var t4=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let t6=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tabIdx=void 0,this.connectors=V.a.state.connectors,this.count=tu.N.state.count,this.filteredCount=tu.N.state.filteredWallets.length,this.isFetchingRecommendedWallets=tu.N.state.isFetchingRecommendedWallets,this.unsubscribe.push(V.a.subscribeKey("connectors",e=>this.connectors=e),tu.N.subscribeKey("count",e=>this.count=e),tu.N.subscribeKey("filteredWallets",e=>this.filteredCount=e.length),tu.N.subscribeKey("isFetchingRecommendedWallets",e=>this.isFetchingRecommendedWallets=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){let e=this.connectors.find(e=>"walletConnect"===e.id),{allWallets:t}=n.H.state;if(!e||"HIDE"===t||"ONLY_MOBILE"===t&&!u.w.isMobile())return null;let i=tu.N.state.featured.length,a=this.count+i,s=a<10?a:10*Math.floor(a/10),l=this.filteredCount>0?this.filteredCount:s,c=`${l}`;this.filteredCount>0?c=`${this.filteredCount}`:l<a&&(c=`${l}+`);let d=X.x.hasAnyConnection(_.o.CONNECTOR_ID.WALLET_CONNECT);return(0,o.qy)`
      <wui-list-wallet
        name="All Wallets"
        walletIcon="allWallets"
        showAllWallets
        @click=${this.onAllWallets.bind(this)}
        tagLabel=${c}
        tagVariant="shade"
        data-testid="all-wallets"
        tabIdx=${(0,r.J)(this.tabIdx)}
        .loading=${this.isFetchingRecommendedWallets}
        loadingSpinnerColor=${this.isFetchingRecommendedWallets?"fg-300":"accent-100"}
        ?disabled=${d}
      ></wui-list-wallet>
    `}onAllWallets(){j.E.sendEvent({type:"track",event:"CLICK_ALL_WALLETS"}),Y.I.push("AllWallets")}};t4([(0,a.MZ)()],t6.prototype,"tabIdx",void 0),t4([(0,a.wk)()],t6.prototype,"connectors",void 0),t4([(0,a.wk)()],t6.prototype,"count",void 0),t4([(0,a.wk)()],t6.prototype,"filteredCount",void 0),t4([(0,a.wk)()],t6.prototype,"isFetchingRecommendedWallets",void 0),t6=t4([(0,p.EM)("w3m-all-wallets-widget")],t6);var t8=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let t7=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tabIdx=void 0,this.connectors=V.a.state.connectors,this.connections=X.x.state.connections,this.unsubscribe.push(V.a.subscribeKey("connectors",e=>this.connectors=e),X.x.subscribeKey("connections",e=>this.connections=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){let e=this.connectors.filter(e=>"ANNOUNCED"===e.type);return e?.length?(0,o.qy)`
      <wui-flex flexDirection="column" gap="xs">
        ${e.filter(e$.g.showConnector).map(e=>{let t=(this.connections.get(e.chain)??[]).some(t=>eG.y.isLowerCaseMatch(t.connectorId,e.id));return(0,o.qy)`
            <wui-list-wallet
              imageSrc=${(0,r.J)(c.$.getConnectorImage(e))}
              name=${e.name??"Unknown"}
              @click=${()=>this.onConnector(e)}
              tagVariant=${t?"shade":"success"}
              tagLabel=${t?"connected":"installed"}
              data-testid=${`wallet-selector-${e.id}`}
              .installed=${!0}
              tabIdx=${(0,r.J)(this.tabIdx)}
            >
            </wui-list-wallet>
          `})}
      </wui-flex>
    `:(this.style.cssText="display: none",null)}onConnector(e){"walletConnect"===e.id?u.w.isMobile()?Y.I.push("AllWallets"):Y.I.push("ConnectingWalletConnect"):Y.I.push("ConnectingExternal",{connector:e})}};t8([(0,a.MZ)()],t7.prototype,"tabIdx",void 0),t8([(0,a.wk)()],t7.prototype,"connectors",void 0),t8([(0,a.wk)()],t7.prototype,"connections",void 0),t7=t8([(0,p.EM)("w3m-connect-announced-widget")],t7);var t9=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ie=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tabIdx=void 0,this.connectors=V.a.state.connectors,this.loading=!1,this.unsubscribe.push(V.a.subscribeKey("connectors",e=>this.connectors=e)),u.w.isTelegram()&&u.w.isIos()&&(this.loading=!X.x.state.wcUri,this.unsubscribe.push(X.x.subscribeKey("wcUri",e=>this.loading=!e)))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){let{customWallets:e}=n.H.state;if(!e?.length)return this.style.cssText="display: none",null;let t=this.filterOutDuplicateWallets(e),i=X.x.hasAnyConnection(_.o.CONNECTOR_ID.WALLET_CONNECT);return(0,o.qy)`<wui-flex flexDirection="column" gap="xs">
      ${t.map(e=>(0,o.qy)`
          <wui-list-wallet
            imageSrc=${(0,r.J)(c.$.getWalletImage(e))}
            name=${e.name??"Unknown"}
            @click=${()=>this.onConnectWallet(e)}
            data-testid=${`wallet-selector-${e.id}`}
            tabIdx=${(0,r.J)(this.tabIdx)}
            ?loading=${this.loading}
            ?disabled=${i}
          >
          </wui-list-wallet>
        `)}
    </wui-flex>`}filterOutDuplicateWallets(e){let t=ea.i.getRecentWallets(),i=this.connectors.map(e=>e.info?.rdns).filter(Boolean),o=t.map(e=>e.rdns).filter(Boolean),a=i.concat(o);if(a.includes("io.metamask.mobile")&&u.w.isMobile()){let e=a.indexOf("io.metamask.mobile");a[e]="io.metamask"}return e.filter(e=>!a.includes(String(e?.rdns)))}onConnectWallet(e){this.loading||Y.I.push("ConnectingWalletConnect",{wallet:e})}};t9([(0,a.MZ)()],ie.prototype,"tabIdx",void 0),t9([(0,a.wk)()],ie.prototype,"connectors",void 0),t9([(0,a.wk)()],ie.prototype,"loading",void 0),ie=t9([(0,p.EM)("w3m-connect-custom-widget")],ie);var it=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ii=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tabIdx=void 0,this.connectors=V.a.state.connectors,this.unsubscribe.push(V.a.subscribeKey("connectors",e=>this.connectors=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){let e=this.connectors.filter(e=>"EXTERNAL"===e.type).filter(e$.g.showConnector).filter(e=>e.id!==_.o.CONNECTOR_ID.COINBASE_SDK);if(!e?.length)return this.style.cssText="display: none",null;let t=X.x.hasAnyConnection(_.o.CONNECTOR_ID.WALLET_CONNECT);return(0,o.qy)`
      <wui-flex flexDirection="column" gap="xs">
        ${e.map(e=>(0,o.qy)`
            <wui-list-wallet
              imageSrc=${(0,r.J)(c.$.getConnectorImage(e))}
              .installed=${!0}
              name=${e.name??"Unknown"}
              data-testid=${`wallet-selector-external-${e.id}`}
              @click=${()=>this.onConnector(e)}
              tabIdx=${(0,r.J)(this.tabIdx)}
              ?disabled=${t}
            >
            </wui-list-wallet>
          `)}
      </wui-flex>
    `}onConnector(e){Y.I.push("ConnectingExternal",{connector:e})}};it([(0,a.MZ)()],ii.prototype,"tabIdx",void 0),it([(0,a.wk)()],ii.prototype,"connectors",void 0),ii=it([(0,p.EM)("w3m-connect-external-widget")],ii);var io=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ia=class extends o.WF{constructor(){super(...arguments),this.tabIdx=void 0,this.wallets=[]}render(){if(!this.wallets.length)return this.style.cssText="display: none",null;let e=X.x.hasAnyConnection(_.o.CONNECTOR_ID.WALLET_CONNECT);return(0,o.qy)`
      <wui-flex flexDirection="column" gap="xs">
        ${this.wallets.map(t=>(0,o.qy)`
            <wui-list-wallet
              data-testid=${`wallet-selector-featured-${t.id}`}
              imageSrc=${(0,r.J)(c.$.getWalletImage(t))}
              name=${t.name??"Unknown"}
              @click=${()=>this.onConnectWallet(t)}
              tabIdx=${(0,r.J)(this.tabIdx)}
              ?disabled=${e}
            >
            </wui-list-wallet>
          `)}
      </wui-flex>
    `}onConnectWallet(e){V.a.selectWalletConnector(e)}};io([(0,a.MZ)()],ia.prototype,"tabIdx",void 0),io([(0,a.MZ)()],ia.prototype,"wallets",void 0),ia=io([(0,p.EM)("w3m-connect-featured-widget")],ia);var ir=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let is=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tabIdx=void 0,this.connectors=[],this.connections=X.x.state.connections,this.unsubscribe.push(X.x.subscribeKey("connections",e=>this.connections=e))}render(){let e=this.connectors.filter(e$.g.showConnector);return 0===e.length?(this.style.cssText="display: none",null):(0,o.qy)`
      <wui-flex flexDirection="column" gap="xs">
        ${e.map(e=>{let t=(this.connections.get(e.chain)??[]).some(t=>eG.y.isLowerCaseMatch(t.connectorId,e.id));return(0,o.qy)`
            <wui-list-wallet
              imageSrc=${(0,r.J)(c.$.getConnectorImage(e))}
              .installed=${!0}
              name=${e.name??"Unknown"}
              tagVariant=${t?"shade":"success"}
              tagLabel=${t?"connected":"installed"}
              data-testid=${`wallet-selector-${e.id}`}
              @click=${()=>this.onConnector(e)}
              tabIdx=${(0,r.J)(this.tabIdx)}
            >
            </wui-list-wallet>
          `})}
      </wui-flex>
    `}onConnector(e){V.a.setActiveConnector(e),Y.I.push("ConnectingExternal",{connector:e})}};ir([(0,a.MZ)()],is.prototype,"tabIdx",void 0),ir([(0,a.MZ)()],is.prototype,"connectors",void 0),ir([(0,a.wk)()],is.prototype,"connections",void 0),is=ir([(0,p.EM)("w3m-connect-injected-widget")],is);var il=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ic=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tabIdx=void 0,this.connectors=V.a.state.connectors,this.unsubscribe.push(V.a.subscribeKey("connectors",e=>this.connectors=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){let e=this.connectors.filter(e=>"MULTI_CHAIN"===e.type&&"WalletConnect"!==e.name);return e?.length?(0,o.qy)`
      <wui-flex flexDirection="column" gap="xs">
        ${e.map(e=>(0,o.qy)`
            <wui-list-wallet
              imageSrc=${(0,r.J)(c.$.getConnectorImage(e))}
              .installed=${!0}
              name=${e.name??"Unknown"}
              tagVariant="shade"
              tagLabel="multichain"
              data-testid=${`wallet-selector-${e.id}`}
              @click=${()=>this.onConnector(e)}
              tabIdx=${(0,r.J)(this.tabIdx)}
            >
            </wui-list-wallet>
          `)}
      </wui-flex>
    `:(this.style.cssText="display: none",null)}onConnector(e){V.a.setActiveConnector(e),Y.I.push("ConnectingMultiChain")}};il([(0,a.MZ)()],ic.prototype,"tabIdx",void 0),il([(0,a.wk)()],ic.prototype,"connectors",void 0),ic=il([(0,p.EM)("w3m-connect-multi-chain-widget")],ic);var id=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let iu=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tabIdx=void 0,this.connectors=V.a.state.connectors,this.loading=!1,this.unsubscribe.push(V.a.subscribeKey("connectors",e=>this.connectors=e)),u.w.isTelegram()&&u.w.isIos()&&(this.loading=!X.x.state.wcUri,this.unsubscribe.push(X.x.subscribeKey("wcUri",e=>this.loading=!e)))}render(){let e=ea.i.getRecentWallets().filter(e=>!tv.A.isExcluded(e)).filter(e=>!this.hasWalletConnector(e)).filter(e=>this.isWalletCompatibleWithCurrentChain(e));if(!e.length)return this.style.cssText="display: none",null;let t=X.x.hasAnyConnection(_.o.CONNECTOR_ID.WALLET_CONNECT);return(0,o.qy)`
      <wui-flex flexDirection="column" gap="xs">
        ${e.map(e=>(0,o.qy)`
            <wui-list-wallet
              imageSrc=${(0,r.J)(c.$.getWalletImage(e))}
              name=${e.name??"Unknown"}
              @click=${()=>this.onConnectWallet(e)}
              tagLabel="recent"
              tagVariant="shade"
              tabIdx=${(0,r.J)(this.tabIdx)}
              ?loading=${this.loading}
              ?disabled=${t}
            >
            </wui-list-wallet>
          `)}
      </wui-flex>
    `}onConnectWallet(e){this.loading||V.a.selectWalletConnector(e)}hasWalletConnector(e){return this.connectors.some(t=>t.id===e.id||t.name===e.name)}isWalletCompatibleWithCurrentChain(e){let t=s.W.state.activeChain;return!t||!e.chains||e.chains.some(e=>t===e.split(":")[0])}};id([(0,a.MZ)()],iu.prototype,"tabIdx",void 0),id([(0,a.wk)()],iu.prototype,"connectors",void 0),id([(0,a.wk)()],iu.prototype,"loading",void 0),iu=id([(0,p.EM)("w3m-connect-recent-widget")],iu);var ih=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ip=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tabIdx=void 0,this.wallets=[],this.loading=!1,u.w.isTelegram()&&u.w.isIos()&&(this.loading=!X.x.state.wcUri,this.unsubscribe.push(X.x.subscribeKey("wcUri",e=>this.loading=!e)))}render(){let{connectors:e}=V.a.state,{customWallets:t,featuredWalletIds:i}=n.H.state,a=ea.i.getRecentWallets(),s=e.find(e=>"walletConnect"===e.id),l=e.filter(e=>"INJECTED"===e.type||"ANNOUNCED"===e.type||"MULTI_CHAIN"===e.type).filter(e=>"Browser Wallet"!==e.name);if(!s)return null;if(i||t||!this.wallets.length)return this.style.cssText="display: none",null;let d=Math.max(0,2-(l.length+a.length)),u=tv.A.filterOutDuplicateWallets(this.wallets).slice(0,d);if(!u.length)return this.style.cssText="display: none",null;let h=X.x.hasAnyConnection(_.o.CONNECTOR_ID.WALLET_CONNECT);return(0,o.qy)`
      <wui-flex flexDirection="column" gap="xs">
        ${u.map(e=>(0,o.qy)`
            <wui-list-wallet
              imageSrc=${(0,r.J)(c.$.getWalletImage(e))}
              name=${e?.name??"Unknown"}
              @click=${()=>this.onConnectWallet(e)}
              tabIdx=${(0,r.J)(this.tabIdx)}
              ?loading=${this.loading}
              ?disabled=${h}
            >
            </wui-list-wallet>
          `)}
      </wui-flex>
    `}onConnectWallet(e){if(this.loading)return;let t=V.a.getConnector({id:e.id,rdns:e.rdns});t?Y.I.push("ConnectingExternal",{connector:t}):Y.I.push("ConnectingWalletConnect",{wallet:e})}};ih([(0,a.MZ)()],ip.prototype,"tabIdx",void 0),ih([(0,a.MZ)()],ip.prototype,"wallets",void 0),ih([(0,a.wk)()],ip.prototype,"loading",void 0),ip=ih([(0,p.EM)("w3m-connect-recommended-widget")],ip);var iw=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ig=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tabIdx=void 0,this.connectors=V.a.state.connectors,this.connectorImages=l.j.state.connectorImages,this.unsubscribe.push(V.a.subscribeKey("connectors",e=>this.connectors=e),l.j.subscribeKey("connectorImages",e=>this.connectorImages=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){if(u.w.isMobile())return this.style.cssText="display: none",null;let e=this.connectors.find(e=>"walletConnect"===e.id);if(!e)return this.style.cssText="display: none",null;let t=e.imageUrl||this.connectorImages[e?.imageId??""],i=X.x.hasAnyConnection(_.o.CONNECTOR_ID.WALLET_CONNECT);return(0,o.qy)`
      <wui-list-wallet
        imageSrc=${(0,r.J)(t)}
        name=${e.name??"Unknown"}
        @click=${()=>this.onConnector(e)}
        tagLabel="qr code"
        tagVariant="main"
        tabIdx=${(0,r.J)(this.tabIdx)}
        data-testid="wallet-selector-walletconnect"
        ?disabled=${i}
      >
      </wui-list-wallet>
    `}onConnector(e){V.a.setActiveConnector(e),Y.I.push("ConnectingWalletConnect")}};iw([(0,a.MZ)()],ig.prototype,"tabIdx",void 0),iw([(0,a.wk)()],ig.prototype,"connectors",void 0),iw([(0,a.wk)()],ig.prototype,"connectorImages",void 0),ig=iw([(0,p.EM)("w3m-connect-walletconnect-widget")],ig);let ib=(0,o.AH)`
  :host {
    margin-top: var(--wui-spacing-3xs);
  }
  wui-separator {
    margin: var(--wui-spacing-m) calc(var(--wui-spacing-m) * -1) var(--wui-spacing-xs)
      calc(var(--wui-spacing-m) * -1);
    width: calc(100% + var(--wui-spacing-s) * 2);
  }
`;var im=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let iv=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.tabIdx=void 0,this.connectors=V.a.state.connectors,this.recommended=tu.N.state.recommended,this.featured=tu.N.state.featured,this.unsubscribe.push(V.a.subscribeKey("connectors",e=>this.connectors=e),tu.N.subscribeKey("recommended",e=>this.recommended=e),tu.N.subscribeKey("featured",e=>this.featured=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){return(0,o.qy)`
      <wui-flex flexDirection="column" gap="xs"> ${this.connectorListTemplate()} </wui-flex>
    `}connectorListTemplate(){let{custom:e,recent:t,announced:i,injected:a,multiChain:n,recommended:s,featured:l,external:c}=e$.g.getConnectorsByType(this.connectors,this.recommended,this.featured);return e$.g.getConnectorTypeOrder({custom:e,recent:t,announced:i,injected:a,multiChain:n,recommended:s,featured:l,external:c}).map(e=>{switch(e){case"injected":return(0,o.qy)`
            ${n.length?(0,o.qy)`<w3m-connect-multi-chain-widget
                  tabIdx=${(0,r.J)(this.tabIdx)}
                ></w3m-connect-multi-chain-widget>`:null}
            ${i.length?(0,o.qy)`<w3m-connect-announced-widget
                  tabIdx=${(0,r.J)(this.tabIdx)}
                ></w3m-connect-announced-widget>`:null}
            ${a.length?(0,o.qy)`<w3m-connect-injected-widget
                  .connectors=${a}
                  tabIdx=${(0,r.J)(this.tabIdx)}
                ></w3m-connect-injected-widget>`:null}
          `;case"walletConnect":return(0,o.qy)`<w3m-connect-walletconnect-widget
            tabIdx=${(0,r.J)(this.tabIdx)}
          ></w3m-connect-walletconnect-widget>`;case"recent":return(0,o.qy)`<w3m-connect-recent-widget
            tabIdx=${(0,r.J)(this.tabIdx)}
          ></w3m-connect-recent-widget>`;case"featured":return(0,o.qy)`<w3m-connect-featured-widget
            .wallets=${l}
            tabIdx=${(0,r.J)(this.tabIdx)}
          ></w3m-connect-featured-widget>`;case"custom":return(0,o.qy)`<w3m-connect-custom-widget
            tabIdx=${(0,r.J)(this.tabIdx)}
          ></w3m-connect-custom-widget>`;case"external":return(0,o.qy)`<w3m-connect-external-widget
            tabIdx=${(0,r.J)(this.tabIdx)}
          ></w3m-connect-external-widget>`;case"recommended":return(0,o.qy)`<w3m-connect-recommended-widget
            .wallets=${s}
            tabIdx=${(0,r.J)(this.tabIdx)}
          ></w3m-connect-recommended-widget>`;default:return console.warn(`Unknown connector type: ${e}`),null}})}};iv.styles=ib,im([(0,a.MZ)()],iv.prototype,"tabIdx",void 0),im([(0,a.wk)()],iv.prototype,"connectors",void 0),im([(0,a.wk)()],iv.prototype,"recommended",void 0),im([(0,a.wk)()],iv.prototype,"featured",void 0),iv=im([(0,p.EM)("w3m-connector-list")],iv);var iy=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ix=class extends o.WF{constructor(){super(...arguments),this.tabIdx=void 0}render(){return(0,o.qy)`
      <wui-flex flexDirection="column" gap="xs">
        <w3m-connector-list tabIdx=${(0,r.J)(this.tabIdx)}></w3m-connector-list>
        <w3m-all-wallets-widget tabIdx=${(0,r.J)(this.tabIdx)}></w3m-all-wallets-widget>
      </wui-flex>
    `}};iy([(0,a.MZ)()],ix.prototype,"tabIdx",void 0),ix=iy([(0,p.EM)("w3m-wallet-login-list")],ix);let ik=(0,o.AH)`
  :host {
    --connect-scroll--top-opacity: 0;
    --connect-scroll--bottom-opacity: 0;
    --connect-mask-image: none;
  }

  .connect {
    max-height: clamp(360px, 470px, 80vh);
    scrollbar-width: none;
    overflow-y: scroll;
    overflow-x: hidden;
    transition: opacity var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: opacity;
    mask-image: var(--connect-mask-image);
  }

  .guide {
    transition: opacity var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: opacity;
  }

  .connect::-webkit-scrollbar {
    display: none;
  }

  .all-wallets {
    flex-flow: column;
  }

  .connect.disabled,
  .guide.disabled {
    opacity: 0.3;
    pointer-events: none;
    user-select: none;
  }

  wui-separator {
    margin: var(--wui-spacing-s) calc(var(--wui-spacing-s) * -1);
    width: calc(100% + var(--wui-spacing-s) * 2);
  }
`;var iC=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let i$=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.connectors=V.a.state.connectors,this.authConnector=this.connectors.find(e=>"AUTH"===e.type),this.features=n.H.state.features,this.remoteFeatures=n.H.state.remoteFeatures,this.enableWallets=n.H.state.enableWallets,this.noAdapters=s.W.state.noAdapters,this.walletGuide="get-started",this.checked=tN.o.state.isLegalCheckboxChecked,this.isEmailEnabled=this.remoteFeatures?.email&&!s.W.state.noAdapters,this.isSocialEnabled=this.remoteFeatures?.socials&&this.remoteFeatures.socials.length>0&&!s.W.state.noAdapters,this.isAuthEnabled=this.checkIfAuthEnabled(this.connectors),this.unsubscribe.push(V.a.subscribeKey("connectors",e=>{this.connectors=e,this.authConnector=this.connectors.find(e=>"AUTH"===e.type),this.isAuthEnabled=this.checkIfAuthEnabled(this.connectors)}),n.H.subscribeKey("features",e=>{this.features=e}),n.H.subscribeKey("remoteFeatures",e=>{this.remoteFeatures=e,this.setEmailAndSocialEnableCheck(this.noAdapters,this.remoteFeatures)}),n.H.subscribeKey("enableWallets",e=>this.enableWallets=e),s.W.subscribeKey("noAdapters",e=>this.setEmailAndSocialEnableCheck(e,this.remoteFeatures)),tN.o.subscribeKey("isLegalCheckboxChecked",e=>this.checked=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e()),this.resizeObserver?.disconnect();let e=this.shadowRoot?.querySelector(".connect");e?.removeEventListener("scroll",this.handleConnectListScroll.bind(this))}firstUpdated(){let e=this.shadowRoot?.querySelector(".connect");e&&(requestAnimationFrame(this.handleConnectListScroll.bind(this)),e?.addEventListener("scroll",this.handleConnectListScroll.bind(this)),this.resizeObserver=new ResizeObserver(()=>{this.handleConnectListScroll()}),this.resizeObserver?.observe(e),this.handleConnectListScroll())}render(){let{termsConditionsUrl:e,privacyPolicyUrl:t}=n.H.state,i=n.H.state.features?.legalCheckbox,a=!!(e||t)&&!!i&&"get-started"===this.walletGuide&&!this.checked,r=n.H.state.enableWalletGuide,s=this.enableWallets,l=this.isSocialEnabled||this.authConnector,c=a?-1:void 0;return(0,o.qy)`
      <wui-flex flexDirection="column">
        ${this.legalCheckboxTemplate()}
        <wui-flex
          data-testid="w3m-connect-scroll-view"
          flexDirection="column"
          class=${(0,ez.H)({connect:!0,disabled:a})}
        >
          <wui-flex
            class="connect-methods"
            flexDirection="column"
            gap="s"
            .padding=${l&&s&&r&&"get-started"===this.walletGuide?["3xs","s","0","s"]:["3xs","s","s","s"]}
          >
            ${this.renderConnectMethod(c)}
          </wui-flex>
        </wui-flex>
        ${this.guideTemplate(a)}
        <w3m-legal-footer></w3m-legal-footer>
      </wui-flex>
    `}setEmailAndSocialEnableCheck(e,t){this.isEmailEnabled=t?.email&&!e,this.isSocialEnabled=t?.socials&&t.socials.length>0&&!e,this.remoteFeatures=t,this.noAdapters=e}checkIfAuthEnabled(e){let t=e.filter(e=>e.type===tq.o.CONNECTOR_TYPE_AUTH).map(e=>e.chain);return _.o.AUTH_CONNECTOR_SUPPORTED_CHAINS.some(e=>t.includes(e))}renderConnectMethod(e){let t=tv.A.getConnectOrderMethod(this.features,this.connectors);return(0,o.qy)`${t.map((t,i)=>{switch(t){case"email":return(0,o.qy)`${this.emailTemplate(e)} ${this.separatorTemplate(i,"email")}`;case"social":return(0,o.qy)`${this.socialListTemplate(e)}
          ${this.separatorTemplate(i,"social")}`;case"wallet":return(0,o.qy)`${this.walletListTemplate(e)}
          ${this.separatorTemplate(i,"wallet")}`;default:return null}})}`}checkMethodEnabled(e){switch(e){case"wallet":return this.enableWallets;case"social":return this.isSocialEnabled&&this.isAuthEnabled;case"email":return this.isEmailEnabled&&this.isAuthEnabled;default:return null}}checkIsThereNextMethod(e){let t=tv.A.getConnectOrderMethod(this.features,this.connectors)[e+1];return t?this.checkMethodEnabled(t)?t:this.checkIsThereNextMethod(e+1):void 0}separatorTemplate(e,t){let i=this.checkIsThereNextMethod(e),a="explore"===this.walletGuide;switch(t){case"wallet":return this.enableWallets&&i&&!a?(0,o.qy)`<wui-separator data-testid="wui-separator" text="or"></wui-separator>`:null;case"email":return this.isAuthEnabled&&this.isEmailEnabled&&"social"!==i&&i?(0,o.qy)`<wui-separator
              data-testid="w3m-email-login-or-separator"
              text="or"
            ></wui-separator>`:null;case"social":return this.isAuthEnabled&&this.isSocialEnabled&&"email"!==i&&i?(0,o.qy)`<wui-separator data-testid="wui-separator" text="or"></wui-separator>`:null;default:return null}}emailTemplate(e){return this.isEmailEnabled&&this.isAuthEnabled?(0,o.qy)`<w3m-email-login-widget
      walletGuide=${this.walletGuide}
      tabIdx=${(0,r.J)(e)}
    ></w3m-email-login-widget>`:null}socialListTemplate(e){return this.isSocialEnabled&&this.isAuthEnabled?(0,o.qy)`<w3m-social-login-widget
      walletGuide=${this.walletGuide}
      tabIdx=${(0,r.J)(e)}
    ></w3m-social-login-widget>`:null}walletListTemplate(e){let t=this.enableWallets,i=this.features?.emailShowWallets===!1,a=this.features?.collapseWallets;return t?(u.w.isTelegram()&&(u.w.isSafari()||u.w.isIos())&&X.x.connectWalletConnect().catch(e=>({})),"explore"===this.walletGuide)?null:this.isAuthEnabled&&(this.isEmailEnabled||this.isSocialEnabled)&&(i||a)?(0,o.qy)`<wui-list-button
        data-testid="w3m-collapse-wallets-button"
        tabIdx=${(0,r.J)(e)}
        @click=${this.onContinueWalletClick.bind(this)}
        text="Continue with a wallet"
      ></wui-list-button>`:(0,o.qy)`<w3m-wallet-login-list tabIdx=${(0,r.J)(e)}></w3m-wallet-login-list>`:null}guideTemplate(e=!1){return n.H.state.enableWalletGuide&&(this.authConnector||this.isSocialEnabled)?(0,o.qy)`
      ${"explore"===this.walletGuide&&!s.W.state.noAdapters?(0,o.qy)`<wui-separator data-testid="wui-separator" id="explore" text="or"></wui-separator>`:null}
      <w3m-wallet-guide
        class=${(0,ez.H)({guide:!0,disabled:e})}
        tabIdx=${(0,r.J)(e?-1:void 0)}
        walletGuide=${this.walletGuide}
      ></w3m-wallet-guide>
    `:null}legalCheckboxTemplate(){return"explore"===this.walletGuide?null:(0,o.qy)`<w3m-legal-checkbox data-testid="w3m-legal-checkbox"></w3m-legal-checkbox>`}handleConnectListScroll(){let e=this.shadowRoot?.querySelector(".connect");e&&(e.scrollHeight>470?(e.style.setProperty("--connect-mask-image",`linear-gradient(
          to bottom,
          rgba(0, 0, 0, calc(1 - var(--connect-scroll--top-opacity))) 0px,
          rgba(200, 200, 200, calc(1 - var(--connect-scroll--top-opacity))) 1px,
          black 40px,
          black calc(100% - 40px),
          rgba(155, 155, 155, calc(1 - var(--connect-scroll--bottom-opacity))) calc(100% - 1px),
          rgba(0, 0, 0, calc(1 - var(--connect-scroll--bottom-opacity))) 100%
        )`),e.style.setProperty("--connect-scroll--top-opacity",p.z8.interpolate([0,50],[0,1],e.scrollTop).toString()),e.style.setProperty("--connect-scroll--bottom-opacity",p.z8.interpolate([0,50],[0,1],e.scrollHeight-e.scrollTop-e.offsetHeight).toString())):(e.style.setProperty("--connect-mask-image","none"),e.style.setProperty("--connect-scroll--top-opacity","0"),e.style.setProperty("--connect-scroll--bottom-opacity","0")))}onContinueWalletClick(){Y.I.push("ConnectWallets")}};i$.styles=ik,iC([(0,tO.w)()],i$.prototype,"connectors",void 0),iC([(0,tO.w)()],i$.prototype,"authConnector",void 0),iC([(0,tO.w)()],i$.prototype,"features",void 0),iC([(0,tO.w)()],i$.prototype,"remoteFeatures",void 0),iC([(0,tO.w)()],i$.prototype,"enableWallets",void 0),iC([(0,tO.w)()],i$.prototype,"noAdapters",void 0),iC([(0,a.MZ)()],i$.prototype,"walletGuide",void 0),iC([(0,tO.w)()],i$.prototype,"checked",void 0),iC([(0,tO.w)()],i$.prototype,"isEmailEnabled",void 0),iC([(0,tO.w)()],i$.prototype,"isSocialEnabled",void 0),iC([(0,tO.w)()],i$.prototype,"isAuthEnabled",void 0),i$=iC([(0,p.EM)("w3m-connect-view")],i$);var iI=i(69148);i(15762),i(34664);let iE=(0,o.AH)`
  wui-flex {
    width: 100%;
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xs);
  }
`;var iS=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let iA=class extends o.WF{constructor(){super(...arguments),this.disabled=!1,this.label="",this.buttonLabel=""}render(){return(0,o.qy)`
      <wui-flex
        justifyContent="space-between"
        alignItems="center"
        .padding=${["1xs","2l","1xs","2l"]}
      >
        <wui-text variant="paragraph-500" color="fg-200">${this.label}</wui-text>
        <wui-chip-button size="sm" variant="shade" text=${this.buttonLabel} icon="chevronRight">
        </wui-chip-button>
      </wui-flex>
    `}};iA.styles=[w.W5,w.fD,iE],iS([(0,a.MZ)({type:Boolean})],iA.prototype,"disabled",void 0),iS([(0,a.MZ)()],iA.prototype,"label",void 0),iS([(0,a.MZ)()],iA.prototype,"buttonLabel",void 0),iA=iS([(0,b.E)("wui-cta-button")],iA);let iW=(0,o.AH)`
  :host {
    display: block;
    padding: 0 var(--wui-spacing-xl) var(--wui-spacing-xl);
  }
`;var iT=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let iR=class extends o.WF{constructor(){super(...arguments),this.wallet=void 0}render(){if(!this.wallet)return this.style.display="none",null;let{name:e,app_store:t,play_store:i,chrome_store:a,homepage:r}=this.wallet,n=u.w.isMobile(),s=u.w.isIos(),l=u.w.isAndroid(),c=[t,i,r,a].filter(Boolean).length>1,d=p.Zv.getTruncateString({string:e,charsStart:12,charsEnd:0,truncate:"end"});return c&&!n?(0,o.qy)`
        <wui-cta-button
          label=${`Don't have ${d}?`}
          buttonLabel="Get"
          @click=${()=>Y.I.push("Downloads",{wallet:this.wallet})}
        ></wui-cta-button>
      `:!c&&r?(0,o.qy)`
        <wui-cta-button
          label=${`Don't have ${d}?`}
          buttonLabel="Get"
          @click=${this.onHomePage.bind(this)}
        ></wui-cta-button>
      `:t&&s?(0,o.qy)`
        <wui-cta-button
          label=${`Don't have ${d}?`}
          buttonLabel="Get"
          @click=${this.onAppStore.bind(this)}
        ></wui-cta-button>
      `:i&&l?(0,o.qy)`
        <wui-cta-button
          label=${`Don't have ${d}?`}
          buttonLabel="Get"
          @click=${this.onPlayStore.bind(this)}
        ></wui-cta-button>
      `:(this.style.display="none",null)}onAppStore(){this.wallet?.app_store&&u.w.openHref(this.wallet.app_store,"_blank")}onPlayStore(){this.wallet?.play_store&&u.w.openHref(this.wallet.play_store,"_blank")}onHomePage(){this.wallet?.homepage&&u.w.openHref(this.wallet.homepage,"_blank")}};iR.styles=[iW],iT([(0,a.MZ)({type:Object})],iR.prototype,"wallet",void 0),iR=iT([(0,p.EM)("w3m-mobile-download-links")],iR);let iO=(0,o.AH)`
  @keyframes shake {
    0% {
      transform: translateX(0);
    }
    25% {
      transform: translateX(3px);
    }
    50% {
      transform: translateX(-3px);
    }
    75% {
      transform: translateX(3px);
    }
    100% {
      transform: translateX(0);
    }
  }

  wui-flex:first-child:not(:only-child) {
    position: relative;
  }

  wui-loading-thumbnail {
    position: absolute;
  }

  wui-icon-box {
    position: absolute;
    right: calc(var(--wui-spacing-3xs) * -1);
    bottom: calc(var(--wui-spacing-3xs) * -1);
    opacity: 0;
    transform: scale(0.5);
    transition-property: opacity, transform;
    transition-duration: var(--wui-duration-lg);
    transition-timing-function: var(--wui-ease-out-power-2);
    will-change: opacity, transform;
  }

  wui-text[align='center'] {
    width: 100%;
    padding: 0px var(--wui-spacing-l);
  }

  [data-error='true'] wui-icon-box {
    opacity: 1;
    transform: scale(1);
  }

  [data-error='true'] > wui-flex:first-child {
    animation: shake 250ms cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
  }

  [data-retry='false'] wui-link {
    display: none;
  }

  [data-retry='true'] wui-link {
    display: block;
    opacity: 1;
  }
`;var iN=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};class iM extends o.WF{constructor(){super(),this.wallet=Y.I.state.data?.wallet,this.connector=Y.I.state.data?.connector,this.timeout=void 0,this.secondaryBtnIcon="refresh",this.onConnect=void 0,this.onRender=void 0,this.onAutoConnect=void 0,this.isWalletConnect=!0,this.unsubscribe=[],this.imageSrc=c.$.getWalletImage(this.wallet)??c.$.getConnectorImage(this.connector),this.name=this.wallet?.name??this.connector?.name??"Wallet",this.isRetrying=!1,this.uri=X.x.state.wcUri,this.error=X.x.state.wcError,this.ready=!1,this.showRetry=!1,this.label=void 0,this.secondaryBtnLabel="Try again",this.secondaryLabel="Accept connection request in the wallet",this.isLoading=!1,this.isMobile=!1,this.onRetry=void 0,this.unsubscribe.push(X.x.subscribeKey("wcUri",e=>{this.uri=e,this.isRetrying&&this.onRetry&&(this.isRetrying=!1,this.onConnect?.())}),X.x.subscribeKey("wcError",e=>this.error=e)),(u.w.isTelegram()||u.w.isSafari())&&u.w.isIos()&&X.x.state.wcUri&&this.onConnect?.()}firstUpdated(){this.onAutoConnect?.(),this.showRetry=!this.onAutoConnect}disconnectedCallback(){this.unsubscribe.forEach(e=>e()),X.x.setWcError(!1),clearTimeout(this.timeout)}render(){this.onRender?.(),this.onShowRetry();let e=this.error?"Connection can be declined if a previous request is still active":this.secondaryLabel,t="";return this.label?t=this.label:(t=`Continue in ${this.name}`,this.error&&(t="Connection declined")),(0,o.qy)`
      <wui-flex
        data-error=${(0,r.J)(this.error)}
        data-retry=${this.showRetry}
        flexDirection="column"
        alignItems="center"
        .padding=${["3xl","xl","xl","xl"]}
        gap="xl"
      >
        <wui-flex justifyContent="center" alignItems="center">
          <wui-wallet-image size="lg" imageSrc=${(0,r.J)(this.imageSrc)}></wui-wallet-image>

          ${this.error?null:this.loaderTemplate()}

          <wui-icon-box
            backgroundColor="error-100"
            background="opaque"
            iconColor="error-100"
            icon="close"
            size="sm"
            border
            borderColor="wui-color-bg-125"
          ></wui-icon-box>
        </wui-flex>

        <wui-flex flexDirection="column" alignItems="center" gap="xs">
          <wui-text
            align="center"
            variant="paragraph-500"
            color=${this.error?"error-100":"fg-100"}
          >
            ${t}
          </wui-text>
          <wui-text align="center" variant="small-500" color="fg-200">${e}</wui-text>
        </wui-flex>

        ${this.secondaryBtnLabel?(0,o.qy)`
              <wui-button
                variant="accent"
                size="md"
                ?disabled=${this.isRetrying||this.isLoading}
                @click=${this.onTryAgain.bind(this)}
                data-testid="w3m-connecting-widget-secondary-button"
              >
                <wui-icon color="inherit" slot="iconLeft" name=${this.secondaryBtnIcon}></wui-icon>
                ${this.secondaryBtnLabel}
              </wui-button>
            `:null}
      </wui-flex>

      ${this.isWalletConnect?(0,o.qy)`
            <wui-flex .padding=${["0","xl","xl","xl"]} justifyContent="center">
              <wui-link @click=${this.onCopyUri} color="fg-200" data-testid="wui-link-copy">
                <wui-icon size="xs" color="fg-200" slot="iconLeft" name="copy"></wui-icon>
                Copy link
              </wui-link>
            </wui-flex>
          `:null}

      <w3m-mobile-download-links .wallet=${this.wallet}></w3m-mobile-download-links>
    `}onShowRetry(){if(this.error&&!this.showRetry){this.showRetry=!0;let e=this.shadowRoot?.querySelector("wui-button");e?.animate([{opacity:0},{opacity:1}],{fill:"forwards",easing:"ease"})}}onTryAgain(){X.x.setWcError(!1),this.onRetry?(this.isRetrying=!0,this.onRetry?.()):this.onConnect?.()}loaderTemplate(){let e=iI.W.state.themeVariables["--w3m-border-radius-master"],t=e?parseInt(e.replace("px",""),10):4;return(0,o.qy)`<wui-loading-thumbnail radius=${9*t}></wui-loading-thumbnail>`}onCopyUri(){try{this.uri&&(u.w.copyToClopboard(this.uri),J.P.showSuccess("Link copied"))}catch{J.P.showError("Failed to copy")}}}iM.styles=iO,iN([(0,a.wk)()],iM.prototype,"isRetrying",void 0),iN([(0,a.wk)()],iM.prototype,"uri",void 0),iN([(0,a.wk)()],iM.prototype,"error",void 0),iN([(0,a.wk)()],iM.prototype,"ready",void 0),iN([(0,a.wk)()],iM.prototype,"showRetry",void 0),iN([(0,a.wk)()],iM.prototype,"label",void 0),iN([(0,a.wk)()],iM.prototype,"secondaryBtnLabel",void 0),iN([(0,a.wk)()],iM.prototype,"secondaryLabel",void 0),iN([(0,a.wk)()],iM.prototype,"isLoading",void 0),iN([(0,a.MZ)({type:Boolean})],iM.prototype,"isMobile",void 0),iN([(0,a.MZ)()],iM.prototype,"onRetry",void 0);let iP=class extends iM{constructor(){if(super(),this.externalViewUnsubscribe=[],this.connectionsByNamespace=X.x.getConnections(this.connector?.chain),this.hasMultipleConnections=this.connectionsByNamespace.length>0,this.remoteFeatures=n.H.state.remoteFeatures,this.currentActiveConnectorId=V.a.state.activeConnectorIds[this.connector?.chain],!this.connector)throw Error("w3m-connecting-view: No connector provided");const e=this.connector?.chain;this.isAlreadyConnected(this.connector)&&(this.secondaryBtnLabel=void 0,this.label=`This account is already linked, change your account in ${this.connector.name}`,this.secondaryLabel=`To link a new account, open ${this.connector.name} and switch to the account you want to link`),j.E.sendEvent({type:"track",event:"SELECT_WALLET",properties:{name:this.connector.name??"Unknown",platform:"browser",displayIndex:this.wallet?.display_index}}),this.onConnect=this.onConnectProxy.bind(this),this.onAutoConnect=this.onConnectProxy.bind(this),this.isWalletConnect=!1,this.externalViewUnsubscribe.push(V.a.subscribeKey("activeConnectorIds",t=>{let i=t[e],o=this.remoteFeatures?.multiWallet;i!==this.currentActiveConnectorId&&(this.hasMultipleConnections&&o?(Y.I.replace("ProfileWallets"),J.P.showSuccess("New Wallet Added")):h.W.close())}),X.x.subscribeKey("connections",this.onConnectionsChange.bind(this)))}disconnectedCallback(){this.externalViewUnsubscribe.forEach(e=>e())}async onConnectProxy(){try{if(this.error=!1,this.connector){if(this.isAlreadyConnected(this.connector))return;this.connector.id===_.o.CONNECTOR_ID.COINBASE_SDK&&this.error||(await X.x.connectExternal(this.connector,this.connector.chain),j.E.sendEvent({type:"track",event:"CONNECT_SUCCESS",properties:{method:"browser",name:this.connector.name||"Unknown"}}))}}catch(e){j.E.sendEvent({type:"track",event:"CONNECT_ERROR",properties:{message:e?.message??"Unknown"}}),this.error=!0}}onConnectionsChange(e){if(this.connector?.chain&&e.get(this.connector.chain)&&this.isAlreadyConnected(this.connector)){let t=e.get(this.connector.chain)??[],i=this.remoteFeatures?.multiWallet;if(0===t.length)Y.I.replace("Connect");else{let e=eZ.b.getConnectionsByConnectorId(this.connectionsByNamespace,this.connector.id).flatMap(e=>e.accounts),o=eZ.b.getConnectionsByConnectorId(t,this.connector.id).flatMap(e=>e.accounts);0===o.length?this.hasMultipleConnections&&i?(Y.I.replace("ProfileWallets"),J.P.showSuccess("Wallet deleted")):h.W.close():!e.every(e=>o.some(t=>eG.y.isLowerCaseMatch(e.address,t.address)))&&i&&Y.I.replace("ProfileWallets")}}}isAlreadyConnected(e){return!!e&&this.connectionsByNamespace.some(t=>eG.y.isLowerCaseMatch(t.connectorId,e.id))}};iP=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-connecting-external-view")],iP);let ij=(0,o.AH)`
  wui-flex,
  wui-list-wallet {
    width: 100%;
  }
`;var iq=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let iD=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.activeConnector=V.a.state.activeConnector,this.unsubscribe.push(V.a.subscribeKey("activeConnector",e=>this.activeConnector=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){return(0,o.qy)`
      <wui-flex
        flexDirection="column"
        alignItems="center"
        .padding=${["m","xl","xl","xl"]}
        gap="xl"
      >
        <wui-flex justifyContent="center" alignItems="center">
          <wui-wallet-image
            size="lg"
            imageSrc=${(0,r.J)(c.$.getConnectorImage(this.activeConnector))}
          ></wui-wallet-image>
        </wui-flex>
        <wui-flex
          flexDirection="column"
          alignItems="center"
          gap="xs"
          .padding=${["0","s","0","s"]}
        >
          <wui-text variant="paragraph-500" color="fg-100">
            Select Chain for ${this.activeConnector?.name}
          </wui-text>
          <wui-text align="center" variant="small-500" color="fg-200"
            >Select which chain to connect to your multi chain wallet</wui-text
          >
        </wui-flex>
        <wui-flex
          flexGrow="1"
          flexDirection="column"
          alignItems="center"
          gap="xs"
          .padding=${["xs","0","xs","0"]}
        >
          ${this.networksTemplate()}
        </wui-flex>
      </wui-flex>
    `}networksTemplate(){return this.activeConnector?.connectors?.map(e=>e.name?(0,o.qy)`
            <wui-list-wallet
              imageSrc=${(0,r.J)(c.$.getChainImage(e.chain))}
              name=${_.o.CHAIN_NAME_MAP[e.chain]}
              @click=${()=>this.onConnector(e)}
              data-testid="wui-list-chain-${e.chain}"
            ></wui-list-wallet>
          `:null)}onConnector(e){let t=this.activeConnector?.connectors?.find(t=>t.chain===e.chain);t?"walletConnect"===t.id?u.w.isMobile()?Y.I.push("AllWallets"):Y.I.push("ConnectingWalletConnect"):Y.I.push("ConnectingExternal",{connector:t}):J.P.showError("Failed to find connector")}};iD.styles=ij,iq([(0,a.wk)()],iD.prototype,"activeConnector",void 0),iD=iq([(0,p.EM)("w3m-connecting-multi-chain-view")],iD);var iL=i(98374),iU=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let iz=class extends o.WF{constructor(){super(...arguments),this.platformTabs=[],this.unsubscribe=[],this.platforms=[],this.onSelectPlatfrom=void 0}disconnectCallback(){this.unsubscribe.forEach(e=>e())}render(){let e=this.generateTabs();return(0,o.qy)`
      <wui-flex justifyContent="center" .padding=${["0","0","l","0"]}>
        <wui-tabs .tabs=${e} .onTabChange=${this.onTabChange.bind(this)}></wui-tabs>
      </wui-flex>
    `}generateTabs(){let e=this.platforms.map(e=>{if("browser"===e)return{label:"Browser",icon:"extension",platform:"browser"};if("mobile"===e)return{label:"Mobile",icon:"mobile",platform:"mobile"};if("qrcode"===e)return{label:"Mobile",icon:"mobile",platform:"qrcode"};if("web"===e)return{label:"Webapp",icon:"browser",platform:"web"};if("desktop"===e)return{label:"Desktop",icon:"desktop",platform:"desktop"};return{label:"Browser",icon:"extension",platform:"unsupported"}});return this.platformTabs=e.map(({platform:e})=>e),e}onTabChange(e){let t=this.platformTabs[e];t&&this.onSelectPlatfrom?.(t)}};iU([(0,a.MZ)({type:Array})],iz.prototype,"platforms",void 0),iU([(0,a.MZ)()],iz.prototype,"onSelectPlatfrom",void 0),iz=iU([(0,p.EM)("w3m-connecting-header")],iz);let iF=class extends iM{constructor(){if(super(),!this.wallet)throw Error("w3m-connecting-wc-browser: No wallet provided");this.onConnect=this.onConnectProxy.bind(this),this.onAutoConnect=this.onConnectProxy.bind(this),j.E.sendEvent({type:"track",event:"SELECT_WALLET",properties:{name:this.wallet.name,platform:"browser",displayIndex:this.wallet?.display_index}})}async onConnectProxy(){try{this.error=!1;let{connectors:e}=V.a.state,t=e.find(e=>"ANNOUNCED"===e.type&&e.info?.rdns===this.wallet?.rdns||"INJECTED"===e.type||e.name===this.wallet?.name);if(t)await X.x.connectExternal(t,t.chain);else throw Error("w3m-connecting-wc-browser: No connector found");h.W.close(),j.E.sendEvent({type:"track",event:"CONNECT_SUCCESS",properties:{method:"browser",name:this.wallet?.name||"Unknown"}})}catch(e){j.E.sendEvent({type:"track",event:"CONNECT_ERROR",properties:{message:e?.message??"Unknown"}}),this.error=!0}}};iF=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-connecting-wc-browser")],iF);let iZ=class extends iM{constructor(){if(super(),!this.wallet)throw Error("w3m-connecting-wc-desktop: No wallet provided");this.onConnect=this.onConnectProxy.bind(this),this.onRender=this.onRenderProxy.bind(this),j.E.sendEvent({type:"track",event:"SELECT_WALLET",properties:{name:this.wallet.name,platform:"desktop",displayIndex:this.wallet?.display_index}})}onRenderProxy(){!this.ready&&this.uri&&(this.ready=!0,this.onConnect?.())}onConnectProxy(){if(this.wallet?.desktop_link&&this.uri)try{this.error=!1;let{desktop_link:e,name:t}=this.wallet,{redirect:i,href:o}=u.w.formatNativeUrl(e,this.uri);X.x.setWcLinking({name:t,href:o}),X.x.setRecentWallet(this.wallet),u.w.openHref(i,"_blank")}catch{this.error=!0}}};iZ=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-connecting-wc-desktop")],iZ);var iB=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let iH=class extends iM{constructor(){if(super(),this.btnLabelTimeout=void 0,this.redirectDeeplink=void 0,this.redirectUniversalLink=void 0,this.target=void 0,this.preferUniversalLinks=n.H.state.experimental_preferUniversalLinks,this.isLoading=!0,this.onConnect=()=>{if(this.wallet?.mobile_link&&this.uri)try{this.error=!1;let{mobile_link:e,link_mode:t,name:i}=this.wallet,{redirect:o,redirectUniversalLink:a,href:r}=u.w.formatNativeUrl(e,this.uri,t);this.redirectDeeplink=o,this.redirectUniversalLink=a,this.target=u.w.isIframe()?"_top":"_self",X.x.setWcLinking({name:i,href:r}),X.x.setRecentWallet(this.wallet),this.preferUniversalLinks&&this.redirectUniversalLink?u.w.openHref(this.redirectUniversalLink,this.target):u.w.openHref(this.redirectDeeplink,this.target)}catch(e){j.E.sendEvent({type:"track",event:"CONNECT_PROXY_ERROR",properties:{message:e instanceof Error?e.message:"Error parsing the deeplink",uri:this.uri,mobile_link:this.wallet.mobile_link,name:this.wallet.name}}),this.error=!0}},!this.wallet)throw Error("w3m-connecting-wc-mobile: No wallet provided");this.secondaryBtnLabel="Open",this.secondaryLabel=K.oU.CONNECT_LABELS.MOBILE,this.secondaryBtnIcon="externalLink",this.onHandleURI(),this.unsubscribe.push(X.x.subscribeKey("wcUri",()=>{this.onHandleURI()})),j.E.sendEvent({type:"track",event:"SELECT_WALLET",properties:{name:this.wallet.name,platform:"mobile",displayIndex:this.wallet?.display_index}})}disconnectedCallback(){super.disconnectedCallback(),clearTimeout(this.btnLabelTimeout)}onHandleURI(){this.isLoading=!this.uri,!this.ready&&this.uri&&(this.ready=!0,this.onConnect?.())}onTryAgain(){X.x.setWcError(!1),this.onConnect?.()}};iB([(0,a.wk)()],iH.prototype,"redirectDeeplink",void 0),iB([(0,a.wk)()],iH.prototype,"redirectUniversalLink",void 0),iB([(0,a.wk)()],iH.prototype,"target",void 0),iB([(0,a.wk)()],iH.prototype,"preferUniversalLinks",void 0),iB([(0,a.wk)()],iH.prototype,"isLoading",void 0),iH=iB([(0,p.EM)("w3m-connecting-wc-mobile")],iH),i(92122),i(69776);let i_=(0,o.AH)`
  @keyframes fadein {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  wui-shimmer {
    width: 100%;
    aspect-ratio: 1 / 1;
    border-radius: clamp(0px, var(--wui-border-radius-l), 40px) !important;
  }

  wui-qr-code {
    opacity: 0;
    animation-duration: 200ms;
    animation-timing-function: ease;
    animation-name: fadein;
    animation-fill-mode: forwards;
  }
`,iV=class extends iM{constructor(){super(),this.forceUpdate=()=>{this.requestUpdate()},window.addEventListener("resize",this.forceUpdate),j.E.sendEvent({type:"track",event:"SELECT_WALLET",properties:{name:this.wallet?.name??"WalletConnect",platform:"qrcode",displayIndex:this.wallet?.display_index}})}disconnectedCallback(){super.disconnectedCallback(),this.unsubscribe?.forEach(e=>e()),window.removeEventListener("resize",this.forceUpdate)}render(){return this.onRenderProxy(),(0,o.qy)`
      <wui-flex
        flexDirection="column"
        alignItems="center"
        .padding=${["0","xl","xl","xl"]}
        gap="xl"
      >
        <wui-shimmer borderRadius="l" width="100%"> ${this.qrCodeTemplate()} </wui-shimmer>

        <wui-text variant="paragraph-500" color="fg-100">
          Scan this QR Code with your phone
        </wui-text>
        ${this.copyTemplate()}
      </wui-flex>
      <w3m-mobile-download-links .wallet=${this.wallet}></w3m-mobile-download-links>
    `}onRenderProxy(){!this.ready&&this.uri&&(this.timeout=setTimeout(()=>{this.ready=!0},200))}qrCodeTemplate(){if(!this.uri||!this.ready)return null;let e=this.getBoundingClientRect().width-40,t=this.wallet?this.wallet.name:void 0;return X.x.setWcLinking(void 0),X.x.setRecentWallet(this.wallet),(0,o.qy)` <wui-qr-code
      size=${e}
      theme=${iI.W.state.themeMode}
      uri=${this.uri}
      imageSrc=${(0,r.J)(c.$.getWalletImage(this.wallet))}
      color=${(0,r.J)(iI.W.state.themeVariables["--w3m-qr-color"])}
      alt=${(0,r.J)(t)}
      data-testid="wui-qr-code"
    ></wui-qr-code>`}copyTemplate(){let e=!this.uri||!this.ready;return(0,o.qy)`<wui-link
      .disabled=${e}
      @click=${this.onCopyUri}
      color="fg-200"
      data-testid="copy-wc2-uri"
    >
      <wui-icon size="xs" color="fg-200" slot="iconLeft" name="copy"></wui-icon>
      Copy link
    </wui-link>`}};iV.styles=i_,iV=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-connecting-wc-qrcode")],iV);let iK=class extends o.WF{constructor(){if(super(),this.wallet=Y.I.state.data?.wallet,!this.wallet)throw Error("w3m-connecting-wc-unsupported: No wallet provided");j.E.sendEvent({type:"track",event:"SELECT_WALLET",properties:{name:this.wallet.name,platform:"browser",displayIndex:this.wallet?.display_index}})}render(){return(0,o.qy)`
      <wui-flex
        flexDirection="column"
        alignItems="center"
        .padding=${["3xl","xl","xl","xl"]}
        gap="xl"
      >
        <wui-wallet-image
          size="lg"
          imageSrc=${(0,r.J)(c.$.getWalletImage(this.wallet))}
        ></wui-wallet-image>

        <wui-text variant="paragraph-500" color="fg-100">Not Detected</wui-text>
      </wui-flex>

      <w3m-mobile-download-links .wallet=${this.wallet}></w3m-mobile-download-links>
    `}};iK=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-connecting-wc-unsupported")],iK);var iJ=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let iG=class extends iM{constructor(){if(super(),this.isLoading=!0,!this.wallet)throw Error("w3m-connecting-wc-web: No wallet provided");this.onConnect=this.onConnectProxy.bind(this),this.secondaryBtnLabel="Open",this.secondaryLabel=K.oU.CONNECT_LABELS.MOBILE,this.secondaryBtnIcon="externalLink",this.updateLoadingState(),this.unsubscribe.push(X.x.subscribeKey("wcUri",()=>{this.updateLoadingState()})),j.E.sendEvent({type:"track",event:"SELECT_WALLET",properties:{name:this.wallet.name,platform:"web",displayIndex:this.wallet?.display_index}})}updateLoadingState(){this.isLoading=!this.uri}onConnectProxy(){if(this.wallet?.webapp_link&&this.uri)try{this.error=!1;let{webapp_link:e,name:t}=this.wallet,{redirect:i,href:o}=u.w.formatUniversalUrl(e,this.uri);X.x.setWcLinking({name:t,href:o}),X.x.setRecentWallet(this.wallet),u.w.openHref(i,"_blank")}catch{this.error=!0}}};iJ([(0,a.wk)()],iG.prototype,"isLoading",void 0),iG=iJ([(0,p.EM)("w3m-connecting-wc-web")],iG);var iY=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let iX=class extends o.WF{constructor(){super(),this.wallet=Y.I.state.data?.wallet,this.unsubscribe=[],this.platform=void 0,this.platforms=[],this.isSiwxEnabled=!!n.H.state.siwx,this.remoteFeatures=n.H.state.remoteFeatures,this.displayBranding=!0,this.determinePlatforms(),this.initializeConnection(),this.unsubscribe.push(n.H.subscribeKey("remoteFeatures",e=>this.remoteFeatures=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){return(0,o.qy)`
      ${this.headerTemplate()}
      <div>${this.platformTemplate()}</div>
      ${this.reownBrandingTemplate()}
    `}reownBrandingTemplate(){return this.remoteFeatures?.reownBranding&&this.displayBranding?(0,o.qy)`<wui-ux-by-reown></wui-ux-by-reown>`:null}async initializeConnection(e=!1){if("browser"!==this.platform&&(!n.H.state.manualWCControl||e))try{let{wcPairingExpiry:t,status:i}=X.x.state;if(e||n.H.state.enableEmbedded||u.w.isPairingExpired(t)||"connecting"===i){let e=X.x.getConnections(s.W.state.activeChain),t=this.remoteFeatures?.multiWallet,i=e.length>0;await X.x.connectWalletConnect({cache:"never"}),this.isSiwxEnabled||(i&&t?(Y.I.replace("ProfileWallets"),J.P.showSuccess("New Wallet Added")):h.W.close())}}catch(e){if(e instanceof Error&&e.message.includes("An error occurred when attempting to switch chain")&&!n.H.state.enableNetworkSwitch&&s.W.state.activeChain){s.W.setActiveCaipNetwork(iL.R.getUnsupportedNetwork(`${s.W.state.activeChain}:${s.W.state.activeCaipNetwork?.id}`)),s.W.showUnsupportedChainUI();return}j.E.sendEvent({type:"track",event:"CONNECT_ERROR",properties:{message:e?.message??"Unknown"}}),X.x.setWcError(!0),J.P.showError(e.message??"Connection error"),X.x.resetWcConnection(),Y.I.goBack()}}determinePlatforms(){if(!this.wallet){this.platforms.push("qrcode"),this.platform="qrcode";return}if(this.platform)return;let{mobile_link:e,desktop_link:t,webapp_link:i,injected:o,rdns:a}=this.wallet,r=o?.map(({injected_id:e})=>e).filter(Boolean),l=[...a?[a]:r??[]],c=!n.H.state.isUniversalProvider&&l.length,d=X.x.checkInstalled(l),h=c&&d,p=t&&!u.w.isMobile();h&&!s.W.state.noAdapters&&this.platforms.push("browser"),e&&this.platforms.push(u.w.isMobile()?"mobile":"qrcode"),i&&this.platforms.push("web"),p&&this.platforms.push("desktop"),h||!c||s.W.state.noAdapters||this.platforms.push("unsupported"),this.platform=this.platforms[0]}platformTemplate(){switch(this.platform){case"browser":return(0,o.qy)`<w3m-connecting-wc-browser></w3m-connecting-wc-browser>`;case"web":return(0,o.qy)`<w3m-connecting-wc-web></w3m-connecting-wc-web>`;case"desktop":return(0,o.qy)`
          <w3m-connecting-wc-desktop .onRetry=${()=>this.initializeConnection(!0)}>
          </w3m-connecting-wc-desktop>
        `;case"mobile":return(0,o.qy)`
          <w3m-connecting-wc-mobile isMobile .onRetry=${()=>this.initializeConnection(!0)}>
          </w3m-connecting-wc-mobile>
        `;case"qrcode":return(0,o.qy)`<w3m-connecting-wc-qrcode></w3m-connecting-wc-qrcode>`;default:return(0,o.qy)`<w3m-connecting-wc-unsupported></w3m-connecting-wc-unsupported>`}}headerTemplate(){return this.platforms.length>1?(0,o.qy)`
      <w3m-connecting-header
        .platforms=${this.platforms}
        .onSelectPlatfrom=${this.onSelectPlatform.bind(this)}
      >
      </w3m-connecting-header>
    `:null}async onSelectPlatform(e){let t=this.shadowRoot?.querySelector("div");t&&(await t.animate([{opacity:1},{opacity:0}],{duration:200,fill:"forwards",easing:"ease"}).finished,this.platform=e,t.animate([{opacity:0},{opacity:1}],{duration:200,fill:"forwards",easing:"ease"}))}};iY([(0,a.wk)()],iX.prototype,"platform",void 0),iY([(0,a.wk)()],iX.prototype,"platforms",void 0),iY([(0,a.wk)()],iX.prototype,"isSiwxEnabled",void 0),iY([(0,a.wk)()],iX.prototype,"remoteFeatures",void 0),iY([(0,a.MZ)({type:Boolean})],iX.prototype,"displayBranding",void 0),iX=iY([(0,p.EM)("w3m-connecting-wc-view")],iX);var iQ=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let i0=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.isMobile=u.w.isMobile(),this.remoteFeatures=n.H.state.remoteFeatures,this.unsubscribe.push(n.H.subscribeKey("remoteFeatures",e=>this.remoteFeatures=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){if(this.isMobile){let{featured:e,recommended:t}=tu.N.state,{customWallets:i}=n.H.state,a=ea.i.getRecentWallets(),r=e.length||t.length||i?.length||a.length;return(0,o.qy)`<wui-flex
        flexDirection="column"
        gap="xs"
        .margin=${["3xs","s","s","s"]}
      >
        ${r?(0,o.qy)`<w3m-connector-list></w3m-connector-list>`:null}
        <w3m-all-wallets-widget></w3m-all-wallets-widget>
      </wui-flex>`}return(0,o.qy)`<wui-flex flexDirection="column" .padding=${["0","0","l","0"]}>
        <w3m-connecting-wc-view .displayBranding=${!1}></w3m-connecting-wc-view>
        <wui-flex flexDirection="column" .padding=${["0","m","0","m"]}>
          <w3m-all-wallets-widget></w3m-all-wallets-widget>
        </wui-flex>
      </wui-flex>
      ${this.reownBrandingTemplate()} `}reownBrandingTemplate(){return this.remoteFeatures?.reownBranding?(0,o.qy)` <wui-flex flexDirection="column" .padding=${["3xs","0","3xs","0"]}>
      <wui-ux-by-reown></wui-ux-by-reown>
    </wui-flex>`:null}};iQ([(0,a.wk)()],i0.prototype,"isMobile",void 0),iQ([(0,a.wk)()],i0.prototype,"remoteFeatures",void 0),i0=iQ([(0,p.EM)("w3m-connecting-wc-basic-view")],i0);var i3=i(39591);let i1=(0,o.AH)`
  .continue-button-container {
    width: 100%;
  }
`;var i2=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let i5=class extends o.WF{constructor(){super(...arguments),this.loading=!1}render(){return(0,o.qy)`
      <wui-flex
        flexDirection="column"
        alignItems="center"
        gap="xxl"
        .padding=${["0","0","l","0"]}
      >
        ${this.onboardingTemplate()} ${this.buttonsTemplate()}
        <wui-link
          @click=${()=>{u.w.openHref(i3.T.URLS.FAQ,"_blank")}}
        >
          Learn more about names
          <wui-icon color="inherit" slot="iconRight" name="externalLink"></wui-icon>
        </wui-link>
      </wui-flex>
    `}onboardingTemplate(){return(0,o.qy)` <wui-flex
      flexDirection="column"
      gap="xxl"
      alignItems="center"
      .padding=${["0","xxl","0","xxl"]}
    >
      <wui-flex gap="s" alignItems="center" justifyContent="center">
        <wui-icon-box
          icon="id"
          size="xl"
          iconSize="xxl"
          iconColor="fg-200"
          backgroundColor="fg-200"
        ></wui-icon-box>
      </wui-flex>
      <wui-flex flexDirection="column" alignItems="center" gap="s">
        <wui-text align="center" variant="medium-600" color="fg-100">
          Choose your account name
        </wui-text>
        <wui-text align="center" variant="paragraph-400" color="fg-100">
          Finally say goodbye to 0x addresses, name your account to make it easier to exchange
          assets
        </wui-text>
      </wui-flex>
    </wui-flex>`}buttonsTemplate(){return(0,o.qy)`<wui-flex
      .padding=${["0","2l","0","2l"]}
      gap="s"
      class="continue-button-container"
    >
      <wui-button
        fullWidth
        .loading=${this.loading}
        size="lg"
        borderRadius="xs"
        @click=${this.handleContinue.bind(this)}
        >Choose name
      </wui-button>
    </wui-flex>`}handleContinue(){Y.I.push("RegisterAccountName"),j.E.sendEvent({type:"track",event:"OPEN_ENS_FLOW",properties:{isSmartAccount:(0,G.lj)(s.W.state.activeChain)===eo.Vl.ACCOUNT_TYPES.SMART_ACCOUNT}})}};i5.styles=i1,i2([(0,a.wk)()],i5.prototype,"loading",void 0),i5=i2([(0,p.EM)("w3m-choose-account-name-view")],i5);let i4=class extends o.WF{constructor(){super(...arguments),this.wallet=Y.I.state.data?.wallet}render(){if(!this.wallet)throw Error("w3m-downloads-view");return(0,o.qy)`
      <wui-flex gap="xs" flexDirection="column" .padding=${["s","s","l","s"]}>
        ${this.chromeTemplate()} ${this.iosTemplate()} ${this.androidTemplate()}
        ${this.homepageTemplate()}
      </wui-flex>
    `}chromeTemplate(){return this.wallet?.chrome_store?(0,o.qy)`<wui-list-item
      variant="icon"
      icon="chromeStore"
      iconVariant="square"
      @click=${this.onChromeStore.bind(this)}
      chevron
    >
      <wui-text variant="paragraph-500" color="fg-100">Chrome Extension</wui-text>
    </wui-list-item>`:null}iosTemplate(){return this.wallet?.app_store?(0,o.qy)`<wui-list-item
      variant="icon"
      icon="appStore"
      iconVariant="square"
      @click=${this.onAppStore.bind(this)}
      chevron
    >
      <wui-text variant="paragraph-500" color="fg-100">iOS App</wui-text>
    </wui-list-item>`:null}androidTemplate(){return this.wallet?.play_store?(0,o.qy)`<wui-list-item
      variant="icon"
      icon="playStore"
      iconVariant="square"
      @click=${this.onPlayStore.bind(this)}
      chevron
    >
      <wui-text variant="paragraph-500" color="fg-100">Android App</wui-text>
    </wui-list-item>`:null}homepageTemplate(){return this.wallet?.homepage?(0,o.qy)`
      <wui-list-item
        variant="icon"
        icon="browser"
        iconVariant="square-blue"
        @click=${this.onHomePage.bind(this)}
        chevron
      >
        <wui-text variant="paragraph-500" color="fg-100">Website</wui-text>
      </wui-list-item>
    `:null}onChromeStore(){this.wallet?.chrome_store&&u.w.openHref(this.wallet.chrome_store,"_blank")}onAppStore(){this.wallet?.app_store&&u.w.openHref(this.wallet.app_store,"_blank")}onPlayStore(){this.wallet?.play_store&&u.w.openHref(this.wallet.play_store,"_blank")}onHomePage(){this.wallet?.homepage&&u.w.openHref(this.wallet.homepage,"_blank")}};i4=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-downloads-view")],i4);let i6=class extends o.WF{render(){return(0,o.qy)`
      <wui-flex flexDirection="column" .padding=${["0","s","s","s"]} gap="xs">
        ${this.recommendedWalletsTemplate()}
        <wui-list-wallet
          name="Explore all"
          showAllWallets
          walletIcon="allWallets"
          icon="externalLink"
          @click=${()=>{u.w.openHref("https://walletconnect.com/explorer?type=wallet","_blank")}}
        ></wui-list-wallet>
      </wui-flex>
    `}recommendedWalletsTemplate(){let{recommended:e,featured:t}=tu.N.state,{customWallets:i}=n.H.state;return[...t,...i??[],...e].slice(0,4).map(e=>(0,o.qy)`
        <wui-list-wallet
          name=${e.name??"Unknown"}
          tagVariant="main"
          imageSrc=${(0,r.J)(c.$.getWalletImage(e))}
          @click=${()=>{u.w.openHref(e.homepage??"https://walletconnect.com/explorer","_blank")}}
        ></wui-list-wallet>
      `)}};i6=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-get-wallet-view")],i6),i(77206);var i8=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let i7=class extends o.WF{constructor(){super(...arguments),this.data=[]}render(){return(0,o.qy)`
      <wui-flex flexDirection="column" alignItems="center" gap="l">
        ${this.data.map(e=>(0,o.qy)`
            <wui-flex flexDirection="column" alignItems="center" gap="xl">
              <wui-flex flexDirection="row" justifyContent="center" gap="1xs">
                ${e.images.map(e=>(0,o.qy)`<wui-visual name=${e}></wui-visual>`)}
              </wui-flex>
            </wui-flex>
            <wui-flex flexDirection="column" alignItems="center" gap="xxs">
              <wui-text variant="paragraph-500" color="fg-100" align="center">
                ${e.title}
              </wui-text>
              <wui-text variant="small-500" color="fg-200" align="center">${e.text}</wui-text>
            </wui-flex>
          `)}
      </wui-flex>
    `}};i8([(0,a.MZ)({type:Array})],i7.prototype,"data",void 0),i7=i8([(0,p.EM)("w3m-help-widget")],i7);let i9=[{images:["login","profile","lock"],title:"One login for all of web3",text:"Log in to any app by connecting your wallet. Say goodbye to countless passwords!"},{images:["defi","nft","eth"],title:"A home for your digital assets",text:"A wallet lets you store, send and receive digital assets like cryptocurrencies and NFTs."},{images:["browser","noun","dao"],title:"Your gateway to a new web",text:"With your wallet, you can explore and interact with DeFi, NFTs, DAOs, and much more."}],oe=class extends o.WF{render(){return(0,o.qy)`
      <wui-flex
        flexDirection="column"
        .padding=${["xxl","xl","xl","xl"]}
        alignItems="center"
        gap="xl"
      >
        <w3m-help-widget .data=${i9}></w3m-help-widget>
        <wui-button variant="main" size="md" @click=${this.onGetWallet.bind(this)}>
          <wui-icon color="inherit" slot="iconLeft" name="wallet"></wui-icon>
          Get a wallet
        </wui-button>
      </wui-flex>
    `}onGetWallet(){j.E.sendEvent({type:"track",event:"CLICK_GET_WALLET"}),Y.I.push("GetWallet")}};oe=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-what-is-a-wallet-view")],oe);let ot=(0,o.AH)`
  wui-flex {
    max-height: clamp(360px, 540px, 80vh);
    overflow: scroll;
    scrollbar-width: none;
    transition: opacity var(--wui-ease-out-power-1) var(--wui-duration-md);
    will-change: opacity;
  }
  wui-flex::-webkit-scrollbar {
    display: none;
  }
  wui-flex.disabled {
    opacity: 0.3;
    pointer-events: none;
    user-select: none;
  }
`;var oi=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let oo=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.checked=tN.o.state.isLegalCheckboxChecked,this.unsubscribe.push(tN.o.subscribeKey("isLegalCheckboxChecked",e=>{this.checked=e}))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){let{termsConditionsUrl:e,privacyPolicyUrl:t}=n.H.state,i=n.H.state.features?.legalCheckbox,a=!!(e||t)&&!!i,s=a&&!this.checked;return(0,o.qy)`
      <w3m-legal-checkbox></w3m-legal-checkbox>
      <wui-flex
        flexDirection="column"
        .padding=${a?["0","s","s","s"]:"s"}
        gap="xs"
        class=${(0,r.J)(s?"disabled":void 0)}
      >
        <w3m-wallet-login-list tabIdx=${(0,r.J)(s?-1:void 0)}></w3m-wallet-login-list>
      </wui-flex>
      <w3m-legal-footer></w3m-legal-footer>
    `}};oo.styles=ot,oi([(0,a.wk)()],oo.prototype,"checked",void 0),oo=oi([(0,p.EM)("w3m-connect-wallets-view")],oo);let oa=(0,o.AH)`
  :host {
    display: block;
    width: var(--wui-box-size-lg);
    height: var(--wui-box-size-lg);
  }

  svg {
    width: var(--wui-box-size-lg);
    height: var(--wui-box-size-lg);
    fill: none;
    stroke: transparent;
    stroke-linecap: round;
  }

  use {
    stroke: var(--wui-color-accent-100);
    stroke-width: 2px;
    stroke-dasharray: 54, 118;
    stroke-dashoffset: 172;
    animation: dash 1s linear infinite;
  }

  @keyframes dash {
    to {
      stroke-dashoffset: 0px;
    }
  }
`,or=class extends o.WF{render(){return(0,o.qy)`
      <svg viewBox="0 0 54 59">
        <path
          id="wui-loader-path"
          d="M17.22 5.295c3.877-2.277 5.737-3.363 7.72-3.726a11.44 11.44 0 0 1 4.12 0c1.983.363 3.844 1.45 7.72 3.726l6.065 3.562c3.876 2.276 5.731 3.372 7.032 4.938a11.896 11.896 0 0 1 2.06 3.63c.683 1.928.688 4.11.688 8.663v7.124c0 4.553-.005 6.735-.688 8.664a11.896 11.896 0 0 1-2.06 3.63c-1.3 1.565-3.156 2.66-7.032 4.937l-6.065 3.563c-3.877 2.276-5.737 3.362-7.72 3.725a11.46 11.46 0 0 1-4.12 0c-1.983-.363-3.844-1.449-7.72-3.726l-6.065-3.562c-3.876-2.276-5.731-3.372-7.032-4.938a11.885 11.885 0 0 1-2.06-3.63c-.682-1.928-.688-4.11-.688-8.663v-7.124c0-4.553.006-6.735.688-8.664a11.885 11.885 0 0 1 2.06-3.63c1.3-1.565 3.156-2.66 7.032-4.937l6.065-3.562Z"
        />
        <use xlink:href="#wui-loader-path"></use>
      </svg>
    `}};or.styles=[w.W5,oa],or=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,b.E)("wui-loading-hexagon")],or),i(12933);let on=(0,o.AH)`
  @keyframes shake {
    0% {
      transform: translateX(0);
    }
    25% {
      transform: translateX(3px);
    }
    50% {
      transform: translateX(-3px);
    }
    75% {
      transform: translateX(3px);
    }
    100% {
      transform: translateX(0);
    }
  }

  wui-flex:first-child:not(:only-child) {
    position: relative;
  }

  wui-loading-hexagon {
    position: absolute;
  }

  wui-icon-box {
    position: absolute;
    right: 4px;
    bottom: 0;
    opacity: 0;
    transform: scale(0.5);
    z-index: 1;
  }

  wui-button {
    display: none;
  }

  [data-error='true'] wui-icon-box {
    opacity: 1;
    transform: scale(1);
  }

  [data-error='true'] > wui-flex:first-child {
    animation: shake 250ms cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
  }

  wui-button[data-retry='true'] {
    display: block;
    opacity: 1;
  }
`;var os=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let ol=class extends o.WF{constructor(){super(),this.network=Y.I.state.data?.network,this.unsubscribe=[],this.showRetry=!1,this.error=!1}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}firstUpdated(){this.onSwitchNetwork()}render(){if(!this.network)throw Error("w3m-network-switch-view: No network provided");this.onShowRetry();let e=this.getLabel(),t=this.getSubLabel();return(0,o.qy)`
      <wui-flex
        data-error=${this.error}
        flexDirection="column"
        alignItems="center"
        .padding=${["3xl","xl","3xl","xl"]}
        gap="xl"
      >
        <wui-flex justifyContent="center" alignItems="center">
          <wui-network-image
            size="lg"
            imageSrc=${(0,r.J)(c.$.getNetworkImage(this.network))}
          ></wui-network-image>

          ${this.error?null:(0,o.qy)`<wui-loading-hexagon></wui-loading-hexagon>`}

          <wui-icon-box
            backgroundColor="error-100"
            background="opaque"
            iconColor="error-100"
            icon="close"
            size="sm"
            ?border=${!0}
            borderColor="wui-color-bg-125"
          ></wui-icon-box>
        </wui-flex>

        <wui-flex flexDirection="column" alignItems="center" gap="xs">
          <wui-text align="center" variant="paragraph-500" color="fg-100">${e}</wui-text>
          <wui-text align="center" variant="small-500" color="fg-200">${t}</wui-text>
        </wui-flex>

        <wui-button
          data-retry=${this.showRetry}
          variant="accent"
          size="md"
          .disabled=${!this.error}
          @click=${this.onSwitchNetwork.bind(this)}
        >
          <wui-icon color="inherit" slot="iconLeft" name="refresh"></wui-icon>
          Try again
        </wui-button>
      </wui-flex>
    `}getSubLabel(){let e=V.a.getConnectorId(s.W.state.activeChain);return V.a.getAuthConnector()&&e===_.o.CONNECTOR_ID.AUTH?"":this.error?"Switch can be declined if chain is not supported by a wallet or previous request is still active":"Accept connection request in your wallet"}getLabel(){let e=V.a.getConnectorId(s.W.state.activeChain);return V.a.getAuthConnector()&&e===_.o.CONNECTOR_ID.AUTH?`Switching to ${this.network?.name??"Unknown"} network...`:this.error?"Switch declined":"Approve in wallet"}onShowRetry(){if(this.error&&!this.showRetry){this.showRetry=!0;let e=this.shadowRoot?.querySelector("wui-button");e?.animate([{opacity:0},{opacity:1}],{fill:"forwards",easing:"ease"})}}async onSwitchNetwork(){try{this.error=!1,s.W.state.activeChain!==this.network?.chainNamespace&&s.W.setIsSwitchingNamespace(!0),this.network&&await s.W.switchActiveNetwork(this.network)}catch(e){this.error=!0}}};ol.styles=on,os([(0,a.wk)()],ol.prototype,"showRetry",void 0),os([(0,a.wk)()],ol.prototype,"error",void 0),ol=os([(0,p.EM)("w3m-network-switch-view")],ol);var oc=i(48075);i(77241),i(80124);let od=(0,o.AH)`
  button {
    column-gap: var(--wui-spacing-s);
    padding: 7px var(--wui-spacing-l) 7px var(--wui-spacing-xs);
    width: 100%;
    transition: all var(--wui-ease-out-power-1) var(--wui-duration-md);
    border-radius: var(--wui-border-radius-xs);
    color: var(--wui-color-fg-100);
  }

  button > wui-text:nth-child(2) {
    display: flex;
    flex: 1;
  }

  button[data-transparent='true'] {
    pointer-events: none;
    background-color: transparent;
  }

  button:hover {
    background-color: var(--wui-color-gray-glass-002);
  }

  button:active {
    background-color: var(--wui-color-gray-glass-005);
  }

  wui-image {
    width: var(--wui-spacing-3xl);
    height: var(--wui-spacing-3xl);
    border-radius: 100%;
  }

  button:disabled {
    background-color: var(--wui-color-gray-glass-002);
    opacity: 0.5;
    cursor: not-allowed;
  }

  button:disabled > wui-tag {
    background-color: var(--wui-color-gray-glass-010);
    color: var(--wui-color-fg-300);
  }
`;var ou=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let oh=class extends o.WF{constructor(){super(...arguments),this.imageSrc="",this.name="",this.disabled=!1,this.selected=!1,this.transparent=!1}render(){return(0,o.qy)`
      <button data-transparent=${this.transparent} ?disabled=${this.disabled}>
        <wui-flex gap="s" alignItems="center">
          ${this.templateNetworkImage()}
          <wui-text variant="paragraph-500" color="inherit">${this.name}</wui-text></wui-flex
        >
        ${this.checkmarkTemplate()}
      </button>
    `}checkmarkTemplate(){return this.selected?(0,o.qy)`<wui-icon size="sm" color="accent-100" name="checkmarkBold"></wui-icon>`:null}templateNetworkImage(){return this.imageSrc?(0,o.qy)`<wui-image size="sm" src=${this.imageSrc} name=${this.name}></wui-image>`:this.imageSrc?null:(0,o.qy)`<wui-network-image
        ?round=${!0}
        size="md"
        name=${this.name}
      ></wui-network-image>`}};oh.styles=[w.W5,w.fD,od],ou([(0,a.MZ)()],oh.prototype,"imageSrc",void 0),ou([(0,a.MZ)()],oh.prototype,"name",void 0),ou([(0,a.MZ)({type:Boolean})],oh.prototype,"disabled",void 0),ou([(0,a.MZ)({type:Boolean})],oh.prototype,"selected",void 0),ou([(0,a.MZ)({type:Boolean})],oh.prototype,"transparent",void 0),oh=ou([(0,b.E)("wui-list-network")],oh);let op=(0,o.AH)`
  .container {
    max-height: 360px;
    overflow: auto;
  }

  .container::-webkit-scrollbar {
    display: none;
  }
`;var ow=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let og=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.network=s.W.state.activeCaipNetwork,this.requestedCaipNetworks=s.W.getCaipNetworks(),this.search="",this.onDebouncedSearch=u.w.debounce(e=>{this.search=e},100),this.unsubscribe.push(l.j.subscribeNetworkImages(()=>this.requestUpdate()),s.W.subscribeKey("activeCaipNetwork",e=>this.network=e),s.W.subscribe(()=>{this.requestedCaipNetworks=s.W.getAllRequestedCaipNetworks()}))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){return(0,o.qy)`
      ${this.templateSearchInput()}
      <wui-flex
        class="container"
        .padding=${["0","s","s","s"]}
        flexDirection="column"
        gap="xs"
      >
        ${this.networksTemplate()}
      </wui-flex>

      <wui-separator></wui-separator>

      <wui-flex padding="s" flexDirection="column" gap="m" alignItems="center">
        <wui-text variant="small-400" color="fg-300" align="center">
          Your connected wallet may not support some of the networks available for this dApp
        </wui-text>
        <wui-link @click=${this.onNetworkHelp.bind(this)}>
          <wui-icon size="xs" color="accent-100" slot="iconLeft" name="helpCircle"></wui-icon>
          What is a network
        </wui-link>
      </wui-flex>
    `}templateSearchInput(){return(0,o.qy)`
      <wui-flex gap="xs" .padding=${["0","s","s","s"]}>
        <wui-input-text
          @inputChange=${this.onInputChange.bind(this)}
          class="network-search-input"
          size="md"
          placeholder="Search network"
          icon="search"
        ></wui-input-text>
      </wui-flex>
    `}onInputChange(e){this.onDebouncedSearch(e.detail)}onNetworkHelp(){j.E.sendEvent({type:"track",event:"CLICK_NETWORK_HELP"}),Y.I.push("WhatIsANetwork")}networksTemplate(){let e=s.W.getAllApprovedCaipNetworkIds(),t=u.w.sortRequestedNetworks(e,this.requestedCaipNetworks);return this.search?this.filteredNetworks=t?.filter(e=>e?.name?.toLowerCase().includes(this.search.toLowerCase())):this.filteredNetworks=t,this.filteredNetworks?.map(e=>(0,o.qy)`
        <wui-list-network
          .selected=${this.network?.id===e.id}
          imageSrc=${(0,r.J)(c.$.getNetworkImage(e))}
          type="network"
          name=${e.name??e.id}
          @click=${()=>this.onSwitchNetwork(e)}
          .disabled=${this.getNetworkDisabled(e)}
          data-testid=${`w3m-network-switch-${e.name??e.id}`}
        ></wui-list-network>
      `)}getNetworkDisabled(e){let t=e.chainNamespace,i=d.U.getCaipAddress(t),o=s.W.getAllApprovedCaipNetworkIds(),a=!1!==s.W.getNetworkProp("supportsAllNetworks",t),r=V.a.getConnectorId(t),n=V.a.getAuthConnector(),l=r===_.o.CONNECTOR_ID.AUTH&&n;return!!i&&!a&&!l&&!o?.includes(e.caipNetworkId)}onSwitchNetwork(e){oc.L.onSwitchNetwork({network:e})}};og.styles=op,ow([(0,a.wk)()],og.prototype,"network",void 0),ow([(0,a.wk)()],og.prototype,"requestedCaipNetworks",void 0),ow([(0,a.wk)()],og.prototype,"filteredNetworks",void 0),ow([(0,a.wk)()],og.prototype,"search",void 0),og=ow([(0,p.EM)("w3m-networks-view")],og);let ob=(0,o.AH)`
  @keyframes shake {
    0% {
      transform: translateX(0);
    }
    25% {
      transform: translateX(3px);
    }
    50% {
      transform: translateX(-3px);
    }
    75% {
      transform: translateX(3px);
    }
    100% {
      transform: translateX(0);
    }
  }

  wui-flex:first-child:not(:only-child) {
    position: relative;
  }

  wui-loading-thumbnail {
    position: absolute;
  }

  wui-visual {
    width: var(--wui-wallet-image-size-lg);
    height: var(--wui-wallet-image-size-lg);
    border-radius: calc(var(--wui-border-radius-5xs) * 9 - var(--wui-border-radius-xxs));
    position: relative;
    overflow: hidden;
  }

  wui-visual::after {
    content: '';
    display: block;
    width: 100%;
    height: 100%;
    position: absolute;
    inset: 0;
    border-radius: calc(var(--wui-border-radius-5xs) * 9 - var(--wui-border-radius-xxs));
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-005);
  }

  wui-icon-box {
    position: absolute;
    right: calc(var(--wui-spacing-3xs) * -1);
    bottom: calc(var(--wui-spacing-3xs) * -1);
    opacity: 0;
    transform: scale(0.5);
    transition:
      opacity var(--wui-ease-out-power-2) var(--wui-duration-lg),
      transform var(--wui-ease-out-power-2) var(--wui-duration-lg);
    will-change: opacity, transform;
  }

  wui-text[align='center'] {
    width: 100%;
    padding: 0px var(--wui-spacing-l);
  }

  [data-error='true'] wui-icon-box {
    opacity: 1;
    transform: scale(1);
  }

  [data-error='true'] > wui-flex:first-child {
    animation: shake 250ms cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
  }

  [data-retry='false'] wui-link {
    display: none;
  }

  [data-retry='true'] wui-link {
    display: block;
    opacity: 1;
  }

  wui-link {
    padding: var(--wui-spacing-4xs) var(--wui-spacing-xxs);
  }

  .capitalize {
    text-transform: capitalize;
  }
`;var of=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let om={eip155:"eth",solana:"solana",bip122:"bitcoin",polkadot:void 0},ov=class extends o.WF{constructor(){super(...arguments),this.unsubscribe=[],this.switchToChain=Y.I.state.data?.switchToChain,this.caipNetwork=Y.I.state.data?.network,this.activeChain=s.W.state.activeChain}firstUpdated(){this.unsubscribe.push(s.W.subscribeKey("activeChain",e=>this.activeChain=e))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){let e=this.switchToChain?_.o.CHAIN_NAME_MAP[this.switchToChain]:"supported";if(!this.switchToChain)return null;let t=_.o.CHAIN_NAME_MAP[this.switchToChain];return(0,o.qy)`
      <wui-flex
        flexDirection="column"
        alignItems="center"
        .padding=${["3xl","xl","xl","xl"]}
        gap="xl"
      >
        <wui-flex justifyContent="center" flexDirection="column" alignItems="center" gap="xl">
          <wui-visual name=${(0,r.J)(om[this.switchToChain])}></wui-visual>
          <wui-text
            data-testid=${`w3m-switch-active-chain-to-${t}`}
            variant="paragraph-500"
            color="fg-100"
            align="center"
            >Switch to <span class="capitalize">${t}</span></wui-text
          >
          <wui-text variant="small-400" color="fg-200" align="center">
            Connected wallet doesn't support connecting to ${e} chain. You
            need to connect with a different wallet.
          </wui-text>
          <wui-button
            data-testid="w3m-switch-active-chain-button"
            size="md"
            @click=${this.switchActiveChain.bind(this)}
            >Switch</wui-button
          >
        </wui-flex>
      </wui-flex>
    `}async switchActiveChain(){this.switchToChain&&(s.W.setIsSwitchingNamespace(!0),V.a.setFilterByNamespace(this.switchToChain),this.caipNetwork?await s.W.switchActiveNetwork(this.caipNetwork):s.W.setActiveNamespace(this.switchToChain),Y.I.reset("Connect"))}};ov.styles=ob,of([(0,a.MZ)()],ov.prototype,"activeChain",void 0),ov=of([(0,p.EM)("w3m-switch-active-chain-view")],ov);let oy=[{images:["network","layers","system"],title:"The system’s nuts and bolts",text:"A network is what brings the blockchain to life, as this technical infrastructure allows apps to access the ledger and smart contract services."},{images:["noun","defiAlt","dao"],title:"Designed for different uses",text:"Each network is designed differently, and may therefore suit certain apps and experiences."}],ox=class extends o.WF{render(){return(0,o.qy)`
      <wui-flex
        flexDirection="column"
        .padding=${["xxl","xl","xl","xl"]}
        alignItems="center"
        gap="xl"
      >
        <w3m-help-widget .data=${oy}></w3m-help-widget>
        <wui-button
          variant="main"
          size="md"
          @click=${()=>{u.w.openHref("https://ethereum.org/en/developers/docs/networks/","_blank")}}
        >
          Learn more
          <wui-icon color="inherit" slot="iconRight" name="externalLink"></wui-icon>
        </wui-button>
      </wui-flex>
    `}};ox=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-what-is-a-network-view")],ox);let ok=(0,o.AH)`
  :host > wui-flex {
    max-height: clamp(360px, 540px, 80vh);
    overflow: scroll;
    scrollbar-width: none;
  }

  :host > wui-flex::-webkit-scrollbar {
    display: none;
  }
`;var oC=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let o$=class extends o.WF{constructor(){super(),this.swapUnsupportedChain=Y.I.state.data?.swapUnsupportedChain,this.unsubscribe=[],this.disconnecting=!1,this.remoteFeatures=n.H.state.remoteFeatures,this.unsubscribe.push(l.j.subscribeNetworkImages(()=>this.requestUpdate()),n.H.subscribeKey("remoteFeatures",e=>{this.remoteFeatures=e}))}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){return(0,o.qy)`
      <wui-flex class="container" flexDirection="column" gap="0">
        <wui-flex
          class="container"
          flexDirection="column"
          .padding=${["m","xl","xs","xl"]}
          alignItems="center"
          gap="xl"
        >
          ${this.descriptionTemplate()}
        </wui-flex>

        <wui-flex flexDirection="column" padding="s" gap="xs">
          ${this.networksTemplate()}
        </wui-flex>

        <wui-separator text="or"></wui-separator>
        <wui-flex flexDirection="column" padding="s" gap="xs">
          <wui-list-item
            variant="icon"
            iconVariant="overlay"
            icon="disconnect"
            ?chevron=${!1}
            .loading=${this.disconnecting}
            @click=${this.onDisconnect.bind(this)}
            data-testid="disconnect-button"
          >
            <wui-text variant="paragraph-500" color="fg-200">Disconnect</wui-text>
          </wui-list-item>
        </wui-flex>
      </wui-flex>
    `}descriptionTemplate(){return this.swapUnsupportedChain?(0,o.qy)`
        <wui-text variant="small-400" color="fg-200" align="center">
          The swap feature doesn’t support your current network. Switch to an available option to
          continue.
        </wui-text>
      `:(0,o.qy)`
      <wui-text variant="small-400" color="fg-200" align="center">
        This app doesn’t support your current network. Switch to an available option to continue.
      </wui-text>
    `}networksTemplate(){let e=s.W.getAllRequestedCaipNetworks(),t=s.W.getAllApprovedCaipNetworkIds(),i=u.w.sortRequestedNetworks(t,e);return(this.swapUnsupportedChain?i.filter(e=>K.oU.SWAP_SUPPORTED_NETWORKS.includes(e.caipNetworkId)):i).map(e=>(0,o.qy)`
        <wui-list-network
          imageSrc=${(0,r.J)(c.$.getNetworkImage(e))}
          name=${e.name??"Unknown"}
          @click=${()=>this.onSwitchNetwork(e)}
        >
        </wui-list-network>
      `)}async onDisconnect(){try{this.disconnecting=!0;let e=s.W.state.activeChain,t=X.x.getConnections(e).length>0,i=e&&V.a.state.activeConnectorIds[e],o=this.remoteFeatures?.multiWallet;await X.x.disconnect(o?{id:i,namespace:e}:{}),t&&o&&(Y.I.push("ProfileWallets"),J.P.showSuccess("Wallet deleted"))}catch{j.E.sendEvent({type:"track",event:"DISCONNECT_ERROR",properties:{message:"Failed to disconnect"}}),J.P.showError("Failed to disconnect")}finally{this.disconnecting=!1}}async onSwitchNetwork(e){let t=d.U.state.caipAddress,i=s.W.getAllApprovedCaipNetworkIds(),o=(s.W.getNetworkProp("supportsAllNetworks",e.chainNamespace),Y.I.state.data);t?i?.includes(e.caipNetworkId)?await s.W.switchActiveNetwork(e):Y.I.push("SwitchNetwork",{...o,network:e}):t||(s.W.setActiveCaipNetwork(e),Y.I.push("Connect"))}};o$.styles=ok,oC([(0,a.wk)()],o$.prototype,"disconnecting",void 0),oC([(0,a.wk)()],o$.prototype,"remoteFeatures",void 0),o$=oC([(0,p.EM)("w3m-unsupported-chain-view")],o$);let oI=(0,o.AH)`
  wui-flex {
    width: 100%;
    background-color: var(--wui-color-gray-glass-005);
    border-radius: var(--wui-border-radius-s);
    padding: var(--wui-spacing-1xs) var(--wui-spacing-s) var(--wui-spacing-1xs)
      var(--wui-spacing-1xs);
  }
`;var oE=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let oS=class extends o.WF{constructor(){super(...arguments),this.icon="externalLink",this.text=""}render(){return(0,o.qy)`
      <wui-flex gap="1xs" alignItems="center">
        <wui-icon-box
          size="sm"
          iconcolor="fg-200"
          backgroundcolor="fg-200"
          icon=${this.icon}
          background="transparent"
        ></wui-icon-box>
        <wui-text variant="small-400" color="fg-200">${this.text}</wui-text>
      </wui-flex>
    `}};oS.styles=[w.W5,w.fD,oI],oE([(0,a.MZ)()],oS.prototype,"icon",void 0),oE([(0,a.MZ)()],oS.prototype,"text",void 0),oS=oE([(0,b.E)("wui-banner")],oS);let oA=(0,o.AH)`
  :host > wui-flex {
    max-height: clamp(360px, 540px, 80vh);
    overflow: scroll;
    scrollbar-width: none;
  }

  :host > wui-flex::-webkit-scrollbar {
    display: none;
  }
`,oW=class extends o.WF{constructor(){super(),this.unsubscribe=[]}disconnectedCallback(){this.unsubscribe.forEach(e=>e())}render(){return(0,o.qy)` <wui-flex
      flexDirection="column"
      .padding=${["xs","s","m","s"]}
      gap="xs"
    >
      <wui-banner
        icon="warningCircle"
        text="You can only receive assets on these networks"
      ></wui-banner>
      ${this.networkTemplate()}
    </wui-flex>`}networkTemplate(){let e=s.W.getAllRequestedCaipNetworks(),t=s.W.getAllApprovedCaipNetworkIds(),i=s.W.state.activeCaipNetwork,a=s.W.checkIfSmartAccountEnabled(),n=u.w.sortRequestedNetworks(t,e);if(a&&(0,G.lj)(i?.chainNamespace)===eo.Vl.ACCOUNT_TYPES.SMART_ACCOUNT){if(!i)return null;n=[i]}return n.filter(e=>e.chainNamespace===i?.chainNamespace).map(e=>(0,o.qy)`
        <wui-list-network
          imageSrc=${(0,r.J)(c.$.getNetworkImage(e))}
          name=${e.name??"Unknown"}
          ?transparent=${!0}
        >
        </wui-list-network>
      `)}};oW.styles=oA,oW=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-wallet-compatible-networks-view")],oW);var oT=i(41300);let oR=(0,o.AH)`
  :host {
    display: flex;
    justify-content: center;
    align-items: center;
    width: var(--wui-icon-box-size-xl);
    height: var(--wui-icon-box-size-xl);
    box-shadow: 0 0 0 8px var(--wui-thumbnail-border);
    border-radius: var(--local-border-radius);
    overflow: hidden;
  }

  wui-icon {
    width: 32px;
    height: 32px;
  }
`;var oO=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let oN=class extends o.WF{render(){return this.style.cssText=`--local-border-radius: ${this.borderRadiusFull?"1000px":"20px"}; background-color: var(--wui-color-modal-bg);`,(0,o.qy)`${this.templateVisual()}`}templateVisual(){return this.imageSrc?(0,o.qy)`<wui-image src=${this.imageSrc} alt=${this.alt??""}></wui-image>`:(0,o.qy)`<wui-icon
      data-parent-size="md"
      size="inherit"
      color="inherit"
      name="walletPlaceholder"
    ></wui-icon>`}};oN.styles=[w.W5,oR],oO([(0,a.MZ)()],oN.prototype,"imageSrc",void 0),oO([(0,a.MZ)()],oN.prototype,"alt",void 0),oO([(0,a.MZ)({type:Boolean})],oN.prototype,"borderRadiusFull",void 0),oN=oO([(0,b.E)("wui-visual-thumbnail")],oN);let oM=(0,o.AH)`
  :host {
    display: flex;
    justify-content: center;
    gap: var(--wui-spacing-2xl);
  }

  wui-visual-thumbnail:nth-child(1) {
    z-index: 1;
  }
`,oP=class extends o.WF{constructor(){super(...arguments),this.dappImageUrl=n.H.state.metadata?.icons,this.walletImageUrl=d.U.state.connectedWalletInfo?.icon}firstUpdated(){let e=this.shadowRoot?.querySelectorAll("wui-visual-thumbnail");e?.[0]&&this.createAnimation(e[0],"translate(18px)"),e?.[1]&&this.createAnimation(e[1],"translate(-18px)")}render(){return(0,o.qy)`
      <wui-visual-thumbnail
        ?borderRadiusFull=${!0}
        .imageSrc=${this.dappImageUrl?.[0]}
      ></wui-visual-thumbnail>
      <wui-visual-thumbnail .imageSrc=${this.walletImageUrl}></wui-visual-thumbnail>
    `}createAnimation(e,t){e.animate([{transform:"translateX(0px)"},{transform:t}],{duration:1600,easing:"cubic-bezier(0.56, 0, 0.48, 1)",direction:"alternate",iterations:1/0})}};oP.styles=oM,oP=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n}([(0,p.EM)("w3m-siwx-sign-message-thumbnails")],oP);var oj=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let oq=class extends o.WF{constructor(){super(...arguments),this.dappName=n.H.state.metadata?.name,this.isCancelling=!1,this.isSigning=!1}render(){return(0,o.qy)`
      <wui-flex justifyContent="center" .padding=${["2xl","0","xxl","0"]}>
        <w3m-siwx-sign-message-thumbnails></w3m-siwx-sign-message-thumbnails>
      </wui-flex>
      <wui-flex
        .padding=${["0","4xl","l","4xl"]}
        gap="s"
        justifyContent="space-between"
      >
        <wui-text variant="paragraph-500" align="center" color="fg-100"
          >${this.dappName??"Dapp"} needs to connect to your wallet</wui-text
        >
      </wui-flex>
      <wui-flex
        .padding=${["0","3xl","l","3xl"]}
        gap="s"
        justifyContent="space-between"
      >
        <wui-text variant="small-400" align="center" color="fg-200"
          >Sign this message to prove you own this wallet and proceed. Canceling will disconnect
          you.</wui-text
        >
      </wui-flex>
      <wui-flex .padding=${["l","xl","xl","xl"]} gap="s" justifyContent="space-between">
        <wui-button
          size="lg"
          borderRadius="xs"
          fullWidth
          variant="neutral"
          ?loading=${this.isCancelling}
          @click=${this.onCancel.bind(this)}
          data-testid="w3m-connecting-siwe-cancel"
        >
          ${this.isCancelling?"Cancelling...":"Cancel"}
        </wui-button>
        <wui-button
          size="lg"
          borderRadius="xs"
          fullWidth
          variant="main"
          @click=${this.onSign.bind(this)}
          ?loading=${this.isSigning}
          data-testid="w3m-connecting-siwe-sign"
        >
          ${this.isSigning?"Signing...":"Sign"}
        </wui-button>
      </wui-flex>
    `}async onSign(){this.isSigning=!0;try{await oT.U.requestSignMessage()}catch(e){if(e instanceof Error&&e.message.includes("OTP is required")){J.P.showError({message:"Something went wrong. We need to verify your account again."}),Y.I.replace("DataCapture");return}throw e}finally{this.isSigning=!1}}async onCancel(){this.isCancelling=!0,await oT.U.cancelSignMessage().finally(()=>this.isCancelling=!1)}};oj([(0,a.wk)()],oq.prototype,"isCancelling",void 0),oj([(0,a.wk)()],oq.prototype,"isSigning",void 0),oq=oj([(0,p.EM)("w3m-siwx-sign-message-view")],oq)},26400:(e,t,i)=>{i(63270)},26478:(e,t,i)=>{var o=i(88088),a=i(53227);i(88364);var r=i(94895),n=i(21736),s=i(69428);let l=(0,o.AH)`
  :host {
    display: block;
    width: var(--local-width);
    height: var(--local-height);
    border-radius: var(--wui-border-radius-3xl);
    box-shadow: 0 0 0 8px var(--wui-color-gray-glass-005);
    overflow: hidden;
    position: relative;
  }

  :host([data-variant='generated']) {
    --mixed-local-color-1: var(--local-color-1);
    --mixed-local-color-2: var(--local-color-2);
    --mixed-local-color-3: var(--local-color-3);
    --mixed-local-color-4: var(--local-color-4);
    --mixed-local-color-5: var(--local-color-5);
  }

  @supports (background: color-mix(in srgb, white 50%, black)) {
    :host([data-variant='generated']) {
      --mixed-local-color-1: color-mix(
        in srgb,
        var(--w3m-color-mix) var(--w3m-color-mix-strength),
        var(--local-color-1)
      );
      --mixed-local-color-2: color-mix(
        in srgb,
        var(--w3m-color-mix) var(--w3m-color-mix-strength),
        var(--local-color-2)
      );
      --mixed-local-color-3: color-mix(
        in srgb,
        var(--w3m-color-mix) var(--w3m-color-mix-strength),
        var(--local-color-3)
      );
      --mixed-local-color-4: color-mix(
        in srgb,
        var(--w3m-color-mix) var(--w3m-color-mix-strength),
        var(--local-color-4)
      );
      --mixed-local-color-5: color-mix(
        in srgb,
        var(--w3m-color-mix) var(--w3m-color-mix-strength),
        var(--local-color-5)
      );
    }
  }

  :host([data-variant='generated']) {
    box-shadow: 0 0 0 8px var(--wui-color-gray-glass-005);
    background: radial-gradient(
      var(--local-radial-circle),
      #fff 0.52%,
      var(--mixed-local-color-5) 31.25%,
      var(--mixed-local-color-3) 51.56%,
      var(--mixed-local-color-2) 65.63%,
      var(--mixed-local-color-1) 82.29%,
      var(--mixed-local-color-4) 100%
    );
  }

  :host([data-variant='default']) {
    box-shadow: 0 0 0 8px var(--wui-color-gray-glass-005);
    background: radial-gradient(
      75.29% 75.29% at 64.96% 24.36%,
      #fff 0.52%,
      #f5ccfc 31.25%,
      #dba4f5 51.56%,
      #9a8ee8 65.63%,
      #6493da 82.29%,
      #6ebdea 100%
    );
  }
`;var c=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let d=class extends o.WF{constructor(){super(...arguments),this.imageSrc=void 0,this.alt=void 0,this.address=void 0,this.size="xl"}render(){return this.style.cssText=`
    --local-width: var(--wui-icon-box-size-${this.size});
    --local-height: var(--wui-icon-box-size-${this.size});
    `,(0,o.qy)`${this.visualTemplate()}`}visualTemplate(){if(this.imageSrc)return this.dataset.variant="image",(0,o.qy)`<wui-image src=${this.imageSrc} alt=${this.alt??"avatar"}></wui-image>`;if(this.address){this.dataset.variant="generated";let e=n.Z.generateAvatarColors(this.address);return this.style.cssText+=`
 ${e}`,null}return this.dataset.variant="default",null}};d.styles=[r.W5,l],c([(0,a.MZ)()],d.prototype,"imageSrc",void 0),c([(0,a.MZ)()],d.prototype,"alt",void 0),c([(0,a.MZ)()],d.prototype,"address",void 0),c([(0,a.MZ)()],d.prototype,"size",void 0),d=c([(0,s.E)("wui-avatar")],d)},29236:(e,t,i)=>{var o=i(88088),a=i(53227);i(19763),i(88364),i(12936);var r=i(94895),n=i(21736),s=i(69428);let l=(0,o.AH)`
  a {
    border: 1px solid var(--wui-color-gray-glass-010);
    border-radius: var(--wui-border-radius-3xl);
  }

  wui-image {
    border-radius: var(--wui-border-radius-3xl);
    overflow: hidden;
  }

  a.disabled > wui-icon:not(.image-icon),
  a.disabled > wui-image {
    filter: grayscale(1);
  }

  a[data-variant='fill'] {
    color: var(--wui-color-inverse-100);
    background-color: var(--wui-color-accent-100);
  }

  a[data-variant='shade'],
  a[data-variant='shadeSmall'] {
    background-color: transparent;
    background-color: var(--wui-color-gray-glass-010);
    color: var(--wui-color-fg-200);
  }

  a[data-variant='success'] {
    column-gap: var(--wui-spacing-xxs);
    border: 1px solid var(--wui-color-success-glass-010);
    background-color: var(--wui-color-success-glass-010);
    color: var(--wui-color-success-100);
  }

  a[data-variant='error'] {
    column-gap: var(--wui-spacing-xxs);
    border: 1px solid var(--wui-color-error-glass-010);
    background-color: var(--wui-color-error-glass-010);
    color: var(--wui-color-error-100);
  }

  a[data-variant='transparent'] {
    column-gap: var(--wui-spacing-xxs);
    background-color: transparent;
    color: var(--wui-color-fg-150);
  }

  a[data-variant='transparent'],
  a[data-variant='success'],
  a[data-variant='shadeSmall'],
  a[data-variant='error'] {
    padding: 7px var(--wui-spacing-s) 7px 10px;
  }

  a[data-variant='transparent']:has(wui-text:first-child),
  a[data-variant='success']:has(wui-text:first-child),
  a[data-variant='shadeSmall']:has(wui-text:first-child),
  a[data-variant='error']:has(wui-text:first-child) {
    padding: 7px var(--wui-spacing-s);
  }

  a[data-variant='fill'],
  a[data-variant='shade'] {
    column-gap: var(--wui-spacing-xs);
    padding: var(--wui-spacing-xxs) var(--wui-spacing-m) var(--wui-spacing-xxs)
      var(--wui-spacing-xs);
  }

  a[data-variant='fill']:has(wui-text:first-child),
  a[data-variant='shade']:has(wui-text:first-child) {
    padding: 9px var(--wui-spacing-m) 9px var(--wui-spacing-m);
  }

  a[data-variant='fill'] > wui-image,
  a[data-variant='shade'] > wui-image {
    width: 24px;
    height: 24px;
  }

  a[data-variant='fill'] > wui-image {
    box-shadow: inset 0 0 0 1px var(--wui-color-accent-090);
  }

  a[data-variant='shade'] > wui-image,
  a[data-variant='shadeSmall'] > wui-image {
    box-shadow: inset 0 0 0 1px var(--wui-color-gray-glass-010);
  }

  a[data-variant='fill'] > wui-icon:not(.image-icon),
  a[data-variant='shade'] > wui-icon:not(.image-icon) {
    width: 14px;
    height: 14px;
  }

  a[data-variant='transparent'] > wui-image,
  a[data-variant='success'] > wui-image,
  a[data-variant='shadeSmall'] > wui-image,
  a[data-variant='error'] > wui-image {
    width: 14px;
    height: 14px;
  }

  a[data-variant='transparent'] > wui-icon:not(.image-icon),
  a[data-variant='success'] > wui-icon:not(.image-icon),
  a[data-variant='shadeSmall'] > wui-icon:not(.image-icon),
  a[data-variant='error'] > wui-icon:not(.image-icon) {
    width: 12px;
    height: 12px;
  }

  a[data-variant='fill']:focus-visible {
    background-color: var(--wui-color-accent-090);
  }

  a[data-variant='shade']:focus-visible,
  a[data-variant='shadeSmall']:focus-visible {
    background-color: var(--wui-color-gray-glass-015);
  }

  a[data-variant='transparent']:focus-visible {
    background-color: var(--wui-color-gray-glass-005);
  }

  a[data-variant='success']:focus-visible {
    background-color: var(--wui-color-success-glass-015);
  }

  a[data-variant='error']:focus-visible {
    background-color: var(--wui-color-error-glass-015);
  }

  a.disabled {
    color: var(--wui-color-gray-glass-015);
    background-color: var(--wui-color-gray-glass-015);
    pointer-events: none;
  }

  @media (hover: hover) and (pointer: fine) {
    a[data-variant='fill']:hover {
      background-color: var(--wui-color-accent-090);
    }

    a[data-variant='shade']:hover,
    a[data-variant='shadeSmall']:hover {
      background-color: var(--wui-color-gray-glass-015);
    }

    a[data-variant='transparent']:hover {
      background-color: var(--wui-color-gray-glass-005);
    }

    a[data-variant='success']:hover {
      background-color: var(--wui-color-success-glass-015);
    }

    a[data-variant='error']:hover {
      background-color: var(--wui-color-error-glass-015);
    }
  }

  a[data-variant='fill']:active {
    background-color: var(--wui-color-accent-080);
  }

  a[data-variant='shade']:active,
  a[data-variant='shadeSmall']:active {
    background-color: var(--wui-color-gray-glass-020);
  }

  a[data-variant='transparent']:active {
    background-color: var(--wui-color-gray-glass-010);
  }

  a[data-variant='success']:active {
    background-color: var(--wui-color-success-glass-020);
  }

  a[data-variant='error']:active {
    background-color: var(--wui-color-error-glass-020);
  }
`;var c=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let d=class extends o.WF{constructor(){super(...arguments),this.variant="fill",this.imageSrc=void 0,this.imageIcon=void 0,this.imageIconSize="md",this.disabled=!1,this.icon="externalLink",this.href="",this.text=void 0}render(){let e="success"===this.variant||"transparent"===this.variant||"shadeSmall"===this.variant;return(0,o.qy)`
      <a
        rel="noreferrer"
        target="_blank"
        href=${this.href}
        class=${this.disabled?"disabled":""}
        data-variant=${this.variant}
      >
        ${this.imageTemplate()}
        <wui-text variant=${e?"small-600":"paragraph-600"} color="inherit">
          ${this.title?this.title:n.Z.getHostName(this.href)}
        </wui-text>
        <wui-icon name=${this.icon} color="inherit" size="inherit"></wui-icon>
      </a>
    `}imageTemplate(){return this.imageSrc?(0,o.qy)`<wui-image src=${this.imageSrc}></wui-image>`:this.imageIcon?(0,o.qy)`<wui-icon
        name=${this.imageIcon}
        color="inherit"
        size=${this.imageIconSize}
        class="image-icon"
      ></wui-icon>`:null}};d.styles=[r.W5,r.fD,l],c([(0,a.MZ)()],d.prototype,"variant",void 0),c([(0,a.MZ)()],d.prototype,"imageSrc",void 0),c([(0,a.MZ)()],d.prototype,"imageIcon",void 0),c([(0,a.MZ)()],d.prototype,"imageIconSize",void 0),c([(0,a.MZ)({type:Boolean})],d.prototype,"disabled",void 0),c([(0,a.MZ)()],d.prototype,"icon",void 0),c([(0,a.MZ)()],d.prototype,"href",void 0),c([(0,a.MZ)()],d.prototype,"text",void 0),d=c([(0,s.E)("wui-chip")],d)},31320:(e,t,i)=>{i.d(t,{a:()=>a});var o=i(88088);let a=(0,o.JW)`<svg  viewBox="0 0 48 54" fill="none">
  <path
    d="M43.4605 10.7248L28.0485 1.61089C25.5438 0.129705 22.4562 0.129705 19.9515 1.61088L4.53951 10.7248C2.03626 12.2051 0.5 14.9365 0.5 17.886V36.1139C0.5 39.0635 2.03626 41.7949 4.53951 43.2752L19.9515 52.3891C22.4562 53.8703 25.5438 53.8703 28.0485 52.3891L43.4605 43.2752C45.9637 41.7949 47.5 39.0635 47.5 36.114V17.8861C47.5 14.9365 45.9637 12.2051 43.4605 10.7248Z"
  />
</svg>`},32297:(e,t,i)=>{var o=i(88088),a=i(53227),r=i(40859);i(12936);var n=i(94895),s=i(69428);i(86212);let l=(0,o.AH)`
  button {
    column-gap: var(--wui-spacing-s);
    padding: 7px var(--wui-spacing-l) 7px var(--wui-spacing-xs);
    width: 100%;
    justify-content: flex-start;
    background-color: var(--wui-color-gray-glass-002);
    border-radius: var(--wui-border-radius-xs);
    color: var(--wui-color-fg-100);
  }

  wui-text {
    text-transform: capitalize;
  }

  wui-text[data-align='left'] {
    display: flex;
    flex: 1;
  }

  wui-text[data-align='center'] {
    display: flex;
    flex: 1;
    justify-content: center;
  }

  .invisible {
    opacity: 0;
    pointer-events: none;
  }

  button:disabled {
    background-color: var(--wui-color-gray-glass-015);
    color: var(--wui-color-gray-glass-015);
  }
`;var c=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let d=class extends o.WF{constructor(){super(...arguments),this.logo="google",this.name="Continue with google",this.align="left",this.disabled=!1}render(){return(0,o.qy)`
      <button ?disabled=${this.disabled} tabindex=${(0,r.J)(this.tabIdx)}>
        <wui-logo logo=${this.logo}></wui-logo>
        <wui-text
          data-align=${this.align}
          variant="paragraph-500"
          color="inherit"
          align=${this.align}
          >${this.name}</wui-text
        >
        ${this.templatePlacement()}
      </button>
    `}templatePlacement(){return"center"===this.align?(0,o.qy)` <wui-logo class="invisible" logo=${this.logo}></wui-logo>`:null}};d.styles=[n.W5,n.fD,l],c([(0,a.MZ)()],d.prototype,"logo",void 0),c([(0,a.MZ)()],d.prototype,"name",void 0),c([(0,a.MZ)()],d.prototype,"align",void 0),c([(0,a.MZ)()],d.prototype,"tabIdx",void 0),c([(0,a.MZ)({type:Boolean})],d.prototype,"disabled",void 0),d=c([(0,s.E)("wui-list-social")],d)},33526:(e,t,i)=>{i.d(t,{y:()=>n});var o=i(34386),a=i(19392),r=i(27510);let n={getTabsByNamespace:e=>e&&e===o.o.CHAIN.EVM?a.H.state.remoteFeatures?.activity===!1?r.o.ACCOUNT_TABS.filter(e=>"Activity"!==e.label):r.o.ACCOUNT_TABS:[],isValidReownName:e=>/^[a-zA-Z0-9]+$/gu.test(e),validateReownName:e=>e.replace(/\^/gu,"").toLowerCase().replace(/[^a-zA-Z0-9]/gu,"")}},34664:(e,t,i)=>{var o=i(88088),a=i(53227);i(19763),i(88364),i(12936);var r=i(94895),n=i(69428);let s=(0,o.AH)`
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
`;var l=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let c=class extends o.WF{constructor(){super(...arguments),this.variant="accent",this.imageSrc="",this.disabled=!1,this.icon="externalLink",this.size="md",this.text=""}render(){let e="sm"===this.size?"small-600":"paragraph-600";return(0,o.qy)`
      <button
        class=${this.disabled?"disabled":""}
        data-variant=${this.variant}
        data-size=${this.size}
        ?disabled=${this.disabled}
      >
        ${this.imageSrc?(0,o.qy)`<wui-image src=${this.imageSrc}></wui-image>`:null}
        <wui-text variant=${e} color="inherit"> ${this.text} </wui-text>
        ${this.icon?(0,o.qy)`<wui-icon name=${this.icon} color="inherit" size="inherit"></wui-icon>`:null}
      </button>
    `}};c.styles=[r.W5,r.fD,s],l([(0,a.MZ)()],c.prototype,"variant",void 0),l([(0,a.MZ)()],c.prototype,"imageSrc",void 0),l([(0,a.MZ)({type:Boolean})],c.prototype,"disabled",void 0),l([(0,a.MZ)()],c.prototype,"icon",void 0),l([(0,a.MZ)()],c.prototype,"size",void 0),l([(0,a.MZ)()],c.prototype,"text",void 0),c=l([(0,n.E)("wui-chip-button")],c)},39591:(e,t,i)=>{i.d(t,{T:()=>o});let o={URLS:{FAQ:"https://walletconnect.com/faq"}}},48657:(e,t,i)=>{var o=i(88088),a=i(53227),r=i(40859);i(12936);var n=i(94895),s=i(69428);i(19493);let l=(0,o.AH)`
  :host {
    position: relative;
    display: inline-block;
  }

  wui-text {
    margin: var(--wui-spacing-xxs) var(--wui-spacing-m) var(--wui-spacing-0) var(--wui-spacing-m);
  }
`;var c=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let d=class extends o.WF{constructor(){super(...arguments),this.disabled=!1}render(){return(0,o.qy)`
      <wui-input-text
        type="email"
        placeholder="Email"
        icon="mail"
        size="mdl"
        .disabled=${this.disabled}
        .value=${this.value}
        data-testid="wui-email-input"
        tabIdx=${(0,r.J)(this.tabIdx)}
      ></wui-input-text>
      ${this.templateError()}
    `}templateError(){return this.errorMessage?(0,o.qy)`<wui-text variant="tiny-500" color="error-100">${this.errorMessage}</wui-text>`:null}};d.styles=[n.W5,l],c([(0,a.MZ)()],d.prototype,"errorMessage",void 0),c([(0,a.MZ)({type:Boolean})],d.prototype,"disabled",void 0),c([(0,a.MZ)()],d.prototype,"value",void 0),c([(0,a.MZ)()],d.prototype,"tabIdx",void 0),d=c([(0,s.E)("wui-email-input")],d)},52589:(e,t,i)=>{var o=i(88088),a=i(53227);i(12936);var r=i(94895),n=i(69428);let s=(0,o.AH)`
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
`;var l=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let c=class extends o.WF{constructor(){super(...arguments),this.text=""}render(){return(0,o.qy)`${this.template()}`}template(){return this.text?(0,o.qy)`<wui-text variant="small-500" color="fg-200">${this.text}</wui-text>`:null}};c.styles=[r.W5,s],l([(0,a.MZ)()],c.prototype,"text",void 0),c=l([(0,n.E)("wui-separator")],c)},78920:(e,t,i)=>{var o=i(88088),a=i(53227),r=i(56726),n=i(44458),s=i(24568),l=i(59306);let c=(0,o.AH)`
  :host {
    width: 100%;
    display: block;
  }
`;var d=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let u=class extends o.WF{constructor(){super(),this.unsubscribe=[],this.text="",this.open=r.I.state.open,this.unsubscribe.push(n.I.subscribeKey("view",()=>{r.I.hide()}),s.W.subscribeKey("open",e=>{e||r.I.hide()}),r.I.subscribeKey("open",e=>{this.open=e}))}disconnectedCallback(){this.unsubscribe.forEach(e=>e()),r.I.hide()}render(){return(0,o.qy)`
      <div
        @pointermove=${this.onMouseEnter.bind(this)}
        @pointerleave=${this.onMouseLeave.bind(this)}
      >
        ${this.renderChildren()}
      </div>
    `}renderChildren(){return(0,o.qy)`<slot></slot> `}onMouseEnter(){let e=this.getBoundingClientRect();this.open||r.I.showTooltip({message:this.text,triggerRect:{width:e.width,height:e.height,left:e.left,top:e.top},variant:"shade"})}onMouseLeave(e){this.contains(e.relatedTarget)||r.I.hide()}};u.styles=[c],d([(0,a.MZ)()],u.prototype,"text",void 0),d([(0,a.wk)()],u.prototype,"open",void 0),u=d([(0,l.EM)("w3m-tooltip-trigger")],u)},80124:(e,t,i)=>{var o=i(88088),a=i(53227);let r=(0,o.JW)`<svg width="86" height="96" fill="none">
  <path
    d="M78.3244 18.926L50.1808 2.45078C45.7376 -0.150261 40.2624 -0.150262 35.8192 2.45078L7.6756 18.926C3.23322 21.5266 0.5 26.3301 0.5 31.5248V64.4752C0.5 69.6699 3.23322 74.4734 7.6756 77.074L35.8192 93.5492C40.2624 96.1503 45.7376 96.1503 50.1808 93.5492L78.3244 77.074C82.7668 74.4734 85.5 69.6699 85.5 64.4752V31.5248C85.5 26.3301 82.7668 21.5266 78.3244 18.926Z"
  />
</svg>`;var n=i(31320);let s=(0,o.JW)`
  <svg fill="none" viewBox="0 0 36 40">
    <path
      d="M15.4 2.1a5.21 5.21 0 0 1 5.2 0l11.61 6.7a5.21 5.21 0 0 1 2.61 4.52v13.4c0 1.87-1 3.59-2.6 4.52l-11.61 6.7c-1.62.93-3.6.93-5.22 0l-11.6-6.7a5.21 5.21 0 0 1-2.61-4.51v-13.4c0-1.87 1-3.6 2.6-4.52L15.4 2.1Z"
    />
  </svg>
`;i(19763),i(88364);var l=i(94895),c=i(69428);let d=(0,o.AH)`
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
`;var u=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let h=class extends o.WF{constructor(){super(...arguments),this.size="md",this.name="uknown",this.networkImagesBySize={sm:s,md:n.a,lg:r},this.selected=!1,this.round=!1}render(){return this.round?(this.dataset.round="true",this.style.cssText=`
      --local-width: var(--wui-spacing-3xl);
      --local-height: var(--wui-spacing-3xl);
      --local-icon-size: var(--wui-spacing-l);
    `):this.style.cssText=`

      --local-path: var(--wui-path-network-${this.size});
      --local-width:  var(--wui-width-network-${this.size});
      --local-height:  var(--wui-height-network-${this.size});
      --local-icon-size:  var(--wui-icon-size-network-${this.size});
    `,(0,o.qy)`${this.templateVisual()} ${this.svgTemplate()} `}svgTemplate(){return this.round?null:this.networkImagesBySize[this.size]}templateVisual(){return this.imageSrc?(0,o.qy)`<wui-image src=${this.imageSrc} alt=${this.name}></wui-image>`:(0,o.qy)`<wui-icon size="inherit" color="fg-200" name="networkPlaceholder"></wui-icon>`}};h.styles=[l.W5,d],u([(0,a.MZ)()],h.prototype,"size",void 0),u([(0,a.MZ)()],h.prototype,"name",void 0),u([(0,a.MZ)({type:Object})],h.prototype,"networkImagesBySize",void 0),u([(0,a.MZ)()],h.prototype,"imageSrc",void 0),u([(0,a.MZ)({type:Boolean})],h.prototype,"selected",void 0),u([(0,a.MZ)({type:Boolean})],h.prototype,"round",void 0),h=u([(0,c.E)("wui-network-image")],h)},86212:(e,t,i)=>{var o=i(88088),a=i(53227);i(19763);var r=i(94895),n=i(69428);let s=(0,o.AH)`
  :host {
    display: flex;
    justify-content: center;
    align-items: center;
    width: 40px;
    height: 40px;
    border-radius: var(--wui-border-radius-3xl);
    border: 1px solid var(--wui-color-gray-glass-005);
    overflow: hidden;
  }

  wui-icon {
    width: 100%;
    height: 100%;
  }
`;var l=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let c=class extends o.WF{constructor(){super(...arguments),this.logo="google"}render(){return(0,o.qy)`<wui-icon color="inherit" size="inherit" name=${this.logo}></wui-icon> `}};c.styles=[r.W5,s],l([(0,a.MZ)()],c.prototype,"logo",void 0),c=l([(0,n.E)("wui-logo")],c)},89373:(e,t,i)=>{i.d(t,{Up:()=>w});var o=i(34386),a=i(34066),r=i(1024),n=i(11446),s=i(43242),l=i(44458),c=i(44963),d=i(37056),u=i(5820);async function h(){l.I.push("ConnectingFarcaster");let e=n.a.getAuthConnector();if(e&&!a.U.state.farcasterUrl)try{let{url:t}=await e.provider.getFarcasterUri();a.U.setFarcasterUrl(t,r.W.state.activeChain)}catch(e){l.I.goBack(),c.P.showError(e)}}async function p(e){l.I.push("ConnectingSocial");let t=n.a.getAuthConnector(),i=null;try{let n=setTimeout(()=>{throw Error("Social login timed out. Please try again.")},45e3);if(t&&e){if(d.w.isTelegram()||(i=function(){try{return d.w.returnOpenHref(`${o.o.SECURE_SITE_SDK_ORIGIN}/loading`,"popupWindow","width=600,height=800,scrollbars=yes")}catch(e){throw Error("Could not open social popup")}}()),i)a.U.setSocialWindow(i,r.W.state.activeChain);else if(!d.w.isTelegram())throw Error("Could not create social popup");let{uri:s}=await t.provider.getSocialRedirectUri({provider:e});if(!s)throw i?.close(),Error("Could not fetch the social redirect uri");if(i&&(i.location.href=s),d.w.isTelegram()){u.i.setTelegramSocialProvider(e);let t=d.w.formatTelegramSocialLoginUrl(s);d.w.openHref(t,"_top")}clearTimeout(n)}}catch(e){i?.close(),c.P.showError(e?.message)}}async function w(e){a.U.setSocialProvider(e,r.W.state.activeChain),s.E.sendEvent({type:"track",event:"SOCIAL_LOGIN_STARTED",properties:{provider:e}}),"farcaster"===e?await h():await p(e)}},92122:(e,t,i)=>{var o=i(88088),a=i(53227);i(19763),i(88364);var r=i(768);function n(e,t,i){return e!==t&&(e-t<0?t-e:e-t)<=i+.1}let s={generate({uri:e,size:t,logoSize:i,dotColor:a="#141414"}){let s,l,c=[],d=(l=Math.sqrt((s=Array.prototype.slice.call(r.create(e,{errorCorrectionLevel:"Q"}).modules.data,0)).length),s.reduce((e,t,i)=>(i%l==0?e.push([t]):e[e.length-1].push(t))&&e,[])),u=t/d.length,h=[{x:0,y:0},{x:1,y:0},{x:0,y:1}];h.forEach(({x:e,y:t})=>{let i=(d.length-7)*u*e,r=(d.length-7)*u*t;for(let e=0;e<h.length;e+=1){let t=u*(7-2*e);c.push((0,o.JW)`
            <rect
              fill=${2===e?a:"transparent"}
              width=${0===e?t-5:t}
              rx= ${0===e?(t-5)*.45:.45*t}
              ry= ${0===e?(t-5)*.45:.45*t}
              stroke=${a}
              stroke-width=${5*(0===e)}
              height=${0===e?t-5:t}
              x= ${0===e?r+u*e+2.5:r+u*e}
              y= ${0===e?i+u*e+2.5:i+u*e}
            />
          `)}});let p=Math.floor((i+25)/u),w=d.length/2-p/2,g=d.length/2+p/2-1,b=[];d.forEach((e,t)=>{e.forEach((e,i)=>{!d[t][i]||t<7&&i<7||t>d.length-8&&i<7||t<7&&i>d.length-8||t>w&&t<g&&i>w&&i<g||b.push([t*u+u/2,i*u+u/2])})});let f={};return b.forEach(([e,t])=>{f[e]?f[e]?.push(t):f[e]=[t]}),Object.entries(f).map(([e,t])=>{let i=t.filter(e=>t.every(t=>!n(e,t,u)));return[Number(e),i]}).forEach(([e,t])=>{t.forEach(t=>{c.push((0,o.JW)`<circle cx=${e} cy=${t} fill=${a} r=${u/2.5} />`)})}),Object.entries(f).filter(([e,t])=>t.length>1).map(([e,t])=>{let i=t.filter(e=>t.some(t=>n(e,t,u)));return[Number(e),i]}).map(([e,t])=>{t.sort((e,t)=>e<t?-1:1);let i=[];for(let e of t){let t=i.find(t=>t.some(t=>n(e,t,u)));t?t.push(e):i.push([e])}return[e,i.map(e=>[e[0],e[e.length-1]])]}).forEach(([e,t])=>{t.forEach(([t,i])=>{c.push((0,o.JW)`
              <line
                x1=${e}
                x2=${e}
                y1=${t}
                y2=${i}
                stroke=${a}
                stroke-width=${u/1.25}
                stroke-linecap="round"
              />
            `)})}),c}};var l=i(94895),c=i(69428);let d=(0,o.AH)`
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
`;var u=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let h=class extends o.WF{constructor(){super(...arguments),this.uri="",this.size=0,this.theme="dark",this.imageSrc=void 0,this.alt=void 0,this.arenaClear=void 0,this.farcaster=void 0}render(){return this.dataset.theme=this.theme,this.dataset.clear=String(this.arenaClear),this.style.cssText=`
     --local-size: ${this.size}px;
     --local-icon-color: ${this.color??"#3396ff"}
    `,(0,o.qy)`${this.templateVisual()} ${this.templateSvg()}`}templateSvg(){let e="light"===this.theme?this.size:this.size-32;return(0,o.JW)`
      <svg height=${e} width=${e}>
        ${s.generate({uri:this.uri,size:e,logoSize:this.arenaClear?0:e/4,dotColor:this.color})}
      </svg>
    `}templateVisual(){return this.imageSrc?(0,o.qy)`<wui-image src=${this.imageSrc} alt=${this.alt??"logo"}></wui-image>`:this.farcaster?(0,o.qy)`<wui-icon
        class="farcaster"
        size="inherit"
        color="inherit"
        name="farcaster"
      ></wui-icon>`:(0,o.qy)`<wui-icon size="inherit" color="inherit" name="walletConnect"></wui-icon>`}};h.styles=[l.W5,d],u([(0,a.MZ)()],h.prototype,"uri",void 0),u([(0,a.MZ)({type:Number})],h.prototype,"size",void 0),u([(0,a.MZ)()],h.prototype,"theme",void 0),u([(0,a.MZ)()],h.prototype,"imageSrc",void 0),u([(0,a.MZ)()],h.prototype,"alt",void 0),u([(0,a.MZ)()],h.prototype,"color",void 0),u([(0,a.MZ)({type:Boolean})],h.prototype,"arenaClear",void 0),u([(0,a.MZ)({type:Boolean})],h.prototype,"farcaster",void 0),h=u([(0,c.E)("wui-qr-code")],h)},92730:(e,t,i)=>{var o=i(88088),a=i(53227);i(19763);var r=i(94895),n=i(69428);let s=(0,o.AH)`
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
`;var l=function(e,t,i,o){var a,r=arguments.length,n=r<3?t:null===o?o=Object.getOwnPropertyDescriptor(t,i):o;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)n=Reflect.decorate(e,t,i,o);else for(var s=e.length-1;s>=0;s--)(a=e[s])&&(n=(r<3?a(n):r>3?a(t,i,n):a(t,i))||n);return r>3&&n&&Object.defineProperty(t,i,n),n};let c=class extends o.WF{constructor(){super(...arguments),this.text="",this.icon="card"}render(){return(0,o.qy)`<button>
      <wui-icon color="accent-100" name=${this.icon} size="lg"></wui-icon>
    </button>`}};c.styles=[r.W5,r.fD,s],l([(0,a.MZ)()],c.prototype,"text",void 0),l([(0,a.MZ)()],c.prototype,"icon",void 0),c=l([(0,n.E)("wui-icon-button")],c)}}]);