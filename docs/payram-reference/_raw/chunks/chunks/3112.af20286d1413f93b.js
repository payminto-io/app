"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[3112],{33112:(e,t,i)=>{i.r(t),i.d(t,{W3mEmailLoginView:()=>k,W3mEmailOtpWidget:()=>h.H,W3mEmailVerifyDeviceView:()=>y,W3mEmailVerifyOtpView:()=>p,W3mUpdateEmailPrimaryOtpView:()=>O,W3mUpdateEmailSecondaryOtpView:()=>C,W3mUpdateEmailWalletView:()=>x});var r=i(1024),a=i(70106),n=i(19392),o=i(43242),l=i(24568),s=i(44458),c=i(44963),d=i(37056),u=i(59306),h=i(59047);let p=class extends h.H{constructor(){super(...arguments),this.onOtpSubmit=async e=>{try{if(this.authConnector){let t=r.W.state.activeChain,i=a.x.getConnections(t),d=n.H.state.remoteFeatures?.multiWallet,u=i.length>0;if(await this.authConnector.provider.connectOtp({otp:e}),o.E.sendEvent({type:"track",event:"EMAIL_VERIFICATION_CODE_PASS"}),t)await a.x.connectExternal(this.authConnector,t);else throw Error("Active chain is not set on ChainControll");if(o.E.sendEvent({type:"track",event:"CONNECT_SUCCESS",properties:{method:"email",name:this.authConnector.name||"Unknown"}}),n.H.state.remoteFeatures?.emailCapture)return;if(n.H.state.siwx)return void l.W.close();if(u&&d){s.I.replace("ProfileWallets"),c.P.showSuccess("New Wallet Added");return}l.W.close()}}catch(e){throw o.E.sendEvent({type:"track",event:"EMAIL_VERIFICATION_CODE_FAIL",properties:{message:d.w.parseError(e)}}),e}},this.onOtpResend=async e=>{this.authConnector&&(await this.authConnector.provider.connectEmail({email:e}),o.E.sendEvent({type:"track",event:"EMAIL_VERIFICATION_CODE_SENT"}))}}};p=function(e,t,i,r){var a,n=arguments.length,o=n<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)o=Reflect.decorate(e,t,i,r);else for(var l=e.length-1;l>=0;l--)(a=e[l])&&(o=(n<3?a(o):n>3?a(t,i,o):a(t,i))||o);return n>3&&o&&Object.defineProperty(t,i,o),o}([(0,u.EM)("w3m-email-verify-otp-view")],p);var m=i(88088),w=i(53227),f=i(11446);i(74970),i(37132),i(45667),i(3658);let E=(0,m.AH)`
  wui-icon-box {
    height: var(--wui-icon-box-size-xl);
    width: var(--wui-icon-box-size-xl);
  }
`;var v=function(e,t,i,r){var a,n=arguments.length,o=n<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)o=Reflect.decorate(e,t,i,r);else for(var l=e.length-1;l>=0;l--)(a=e[l])&&(o=(n<3?a(o):n>3?a(t,i,o):a(t,i))||o);return n>3&&o&&Object.defineProperty(t,i,o),o};let y=class extends m.WF{constructor(){super(),this.email=s.I.state.data?.email,this.authConnector=f.a.getAuthConnector(),this.loading=!1,this.listenForDeviceApproval()}render(){if(!this.email)throw Error("w3m-email-verify-device-view: No email provided");if(!this.authConnector)throw Error("w3m-email-verify-device-view: No auth connector provided");return(0,m.qy)`
      <wui-flex
        flexDirection="column"
        alignItems="center"
        .padding=${["xxl","s","xxl","s"]}
        gap="l"
      >
        <wui-icon-box
          size="xl"
          iconcolor="accent-100"
          backgroundcolor="accent-100"
          icon="verify"
          background="opaque"
        ></wui-icon-box>

        <wui-flex flexDirection="column" alignItems="center" gap="s">
          <wui-flex flexDirection="column" alignItems="center">
            <wui-text variant="paragraph-400" color="fg-100">
              Approve the login link we sent to
            </wui-text>
            <wui-text variant="paragraph-400" color="fg-100"><b>${this.email}</b></wui-text>
          </wui-flex>

          <wui-text variant="small-400" color="fg-200" align="center">
            The code expires in 20 minutes
          </wui-text>

          <wui-flex alignItems="center" id="w3m-resend-section" gap="xs">
            <wui-text variant="small-400" color="fg-100" align="center">
              Didn't receive it?
            </wui-text>
            <wui-link @click=${this.onResendCode.bind(this)} .disabled=${this.loading}>
              Resend email
            </wui-link>
          </wui-flex>
        </wui-flex>
      </wui-flex>
    `}async listenForDeviceApproval(){if(this.authConnector)try{await this.authConnector.provider.connectDevice(),o.E.sendEvent({type:"track",event:"DEVICE_REGISTERED_FOR_EMAIL"}),o.E.sendEvent({type:"track",event:"EMAIL_VERIFICATION_CODE_SENT"}),s.I.replace("EmailVerifyOtp",{email:this.email})}catch(e){s.I.goBack()}}async onResendCode(){try{if(!this.loading){if(!this.authConnector||!this.email)throw Error("w3m-email-login-widget: Unable to resend email");this.loading=!0,await this.authConnector.provider.connectEmail({email:this.email}),this.listenForDeviceApproval(),c.P.showSuccess("Code email resent")}}catch(e){c.P.showError(e)}finally{this.loading=!1}}};y.styles=E,v([(0,w.wk)()],y.prototype,"loading",void 0),y=v([(0,u.EM)("w3m-email-verify-device-view")],y);var g=i(22096);i(66449),i(48657);let b=(0,m.AH)`
  wui-email-input {
    width: 100%;
  }

  form {
    width: 100%;
    display: block;
    position: relative;
  }
`;var I=function(e,t,i,r){var a,n=arguments.length,o=n<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)o=Reflect.decorate(e,t,i,r);else for(var l=e.length-1;l>=0;l--)(a=e[l])&&(o=(n<3?a(o):n>3?a(t,i,o):a(t,i))||o);return n>3&&o&&Object.defineProperty(t,i,o),o};let x=class extends m.WF{constructor(){super(...arguments),this.formRef=(0,g._)(),this.initialEmail=s.I.state.data?.email??"",this.redirectView=s.I.state.data?.redirectView,this.email="",this.loading=!1}firstUpdated(){this.formRef.value?.addEventListener("keydown",e=>{"Enter"===e.key&&this.onSubmitEmail(e)})}render(){return(0,m.qy)`
      <wui-flex flexDirection="column" padding="m" gap="m">
        <form ${(0,g.K)(this.formRef)} @submit=${this.onSubmitEmail.bind(this)}>
          <wui-email-input
            value=${this.initialEmail}
            .disabled=${this.loading}
            @inputChange=${this.onEmailInputChange.bind(this)}
          >
          </wui-email-input>
          <input type="submit" hidden />
        </form>
        ${this.buttonsTemplate()}
      </wui-flex>
    `}onEmailInputChange(e){this.email=e.detail}async onSubmitEmail(e){try{if(this.loading)return;this.loading=!0,e.preventDefault();let t=f.a.getAuthConnector();if(!t)throw Error("w3m-update-email-wallet: Auth connector not found");let i=await t.provider.updateEmail({email:this.email});o.E.sendEvent({type:"track",event:"EMAIL_EDIT"}),"VERIFY_SECONDARY_OTP"===i.action?s.I.push("UpdateEmailSecondaryOtp",{email:this.initialEmail,newEmail:this.email,redirectView:this.redirectView}):s.I.push("UpdateEmailPrimaryOtp",{email:this.initialEmail,newEmail:this.email,redirectView:this.redirectView})}catch(e){c.P.showError(e),this.loading=!1}}buttonsTemplate(){let e=!this.loading&&this.email.length>3&&this.email!==this.initialEmail;return this.redirectView?(0,m.qy)`
      <wui-flex gap="s">
        <wui-button size="md" variant="neutral" fullWidth @click=${s.I.goBack}>
          Cancel
        </wui-button>

        <wui-button
          size="md"
          variant="main"
          fullWidth
          @click=${this.onSubmitEmail.bind(this)}
          .disabled=${!e}
          .loading=${this.loading}
        >
          Save
        </wui-button>
      </wui-flex>
    `:(0,m.qy)`
        <wui-button
          size="md"
          variant="main"
          fullWidth
          @click=${this.onSubmitEmail.bind(this)}
          .disabled=${!e}
          .loading=${this.loading}
        >
          Save
        </wui-button>
      `}};x.styles=b,I([(0,w.wk)()],x.prototype,"email",void 0),I([(0,w.wk)()],x.prototype,"loading",void 0),x=I([(0,u.EM)("w3m-update-email-wallet-view")],x);let O=class extends h.H{constructor(){super(),this.email=s.I.state.data?.email,this.onOtpSubmit=async e=>{try{this.authConnector&&(await this.authConnector.provider.updateEmailPrimaryOtp({otp:e}),o.E.sendEvent({type:"track",event:"EMAIL_VERIFICATION_CODE_PASS"}),s.I.replace("UpdateEmailSecondaryOtp",s.I.state.data))}catch(e){throw o.E.sendEvent({type:"track",event:"EMAIL_VERIFICATION_CODE_FAIL",properties:{message:d.w.parseError(e)}}),e}},this.onStartOver=()=>{s.I.replace("UpdateEmailWallet",s.I.state.data)}}};O=function(e,t,i,r){var a,n=arguments.length,o=n<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)o=Reflect.decorate(e,t,i,r);else for(var l=e.length-1;l>=0;l--)(a=e[l])&&(o=(n<3?a(o):n>3?a(t,i,o):a(t,i))||o);return n>3&&o&&Object.defineProperty(t,i,o),o}([(0,u.EM)("w3m-update-email-primary-otp-view")],O);let C=class extends h.H{constructor(){super(),this.email=s.I.state.data?.newEmail,this.redirectView=s.I.state.data?.redirectView,this.onOtpSubmit=async e=>{try{this.authConnector&&(await this.authConnector.provider.updateEmailSecondaryOtp({otp:e}),o.E.sendEvent({type:"track",event:"EMAIL_VERIFICATION_CODE_PASS"}),this.redirectView&&s.I.reset(this.redirectView))}catch(e){throw o.E.sendEvent({type:"track",event:"EMAIL_VERIFICATION_CODE_FAIL",properties:{message:d.w.parseError(e)}}),e}},this.onStartOver=()=>{s.I.replace("UpdateEmailWallet",s.I.state.data)}}};C=function(e,t,i,r){var a,n=arguments.length,o=n<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)o=Reflect.decorate(e,t,i,r);else for(var l=e.length-1;l>=0;l--)(a=e[l])&&(o=(n<3?a(o):n>3?a(t,i,o):a(t,i))||o);return n>3&&o&&Object.defineProperty(t,i,o),o}([(0,u.EM)("w3m-update-email-secondary-otp-view")],C);var A=i(34386),R=i(13226),_=function(e,t,i,r){var a,n=arguments.length,o=n<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)o=Reflect.decorate(e,t,i,r);else for(var l=e.length-1;l>=0;l--)(a=e[l])&&(o=(n<3?a(o):n>3?a(t,i,o):a(t,i))||o);return n>3&&o&&Object.defineProperty(t,i,o),o};let k=class extends m.WF{constructor(){super(),this.authConnector=f.a.getAuthConnector(),this.isEmailEnabled=n.H.state.remoteFeatures?.email,this.isAuthEnabled=this.checkIfAuthEnabled(f.a.state.connectors),this.connectors=f.a.state.connectors,f.a.subscribeKey("connectors",e=>{this.connectors=e,this.isAuthEnabled=this.checkIfAuthEnabled(this.connectors)})}render(){if(!this.isEmailEnabled)throw Error("w3m-email-login-view: Email is not enabled");if(!this.isAuthEnabled)throw Error("w3m-email-login-view: No auth connector provided");return(0,m.qy)`<wui-flex
      flexDirection="column"
      .padding=${["3xs","m","m","m"]}
      gap="l"
    >
      <w3m-email-login-widget></w3m-email-login-widget>
    </wui-flex> `}checkIfAuthEnabled(e){let t=e.filter(e=>e.type===R.o.CONNECTOR_TYPE_AUTH).map(e=>e.chain);return A.o.AUTH_CONNECTOR_SUPPORTED_CHAINS.some(e=>t.includes(e))}};_([(0,w.wk)()],k.prototype,"connectors",void 0),k=_([(0,u.EM)("w3m-email-login-view")],k)},37132:(e,t,i)=>{i(15642)},48657:(e,t,i)=>{var r=i(88088),a=i(53227),n=i(40859);i(12936);var o=i(94895),l=i(69428);i(19493);let s=(0,r.AH)`
  :host {
    position: relative;
    display: inline-block;
  }

  wui-text {
    margin: var(--wui-spacing-xxs) var(--wui-spacing-m) var(--wui-spacing-0) var(--wui-spacing-m);
  }
`;var c=function(e,t,i,r){var a,n=arguments.length,o=n<3?t:null===r?r=Object.getOwnPropertyDescriptor(t,i):r;if("object"==typeof Reflect&&"function"==typeof Reflect.decorate)o=Reflect.decorate(e,t,i,r);else for(var l=e.length-1;l>=0;l--)(a=e[l])&&(o=(n<3?a(o):n>3?a(t,i,o):a(t,i))||o);return n>3&&o&&Object.defineProperty(t,i,o),o};let d=class extends r.WF{constructor(){super(...arguments),this.disabled=!1}render(){return(0,r.qy)`
      <wui-input-text
        type="email"
        placeholder="Email"
        icon="mail"
        size="mdl"
        .disabled=${this.disabled}
        .value=${this.value}
        data-testid="wui-email-input"
        tabIdx=${(0,n.J)(this.tabIdx)}
      ></wui-input-text>
      ${this.templateError()}
    `}templateError(){return this.errorMessage?(0,r.qy)`<wui-text variant="tiny-500" color="error-100">${this.errorMessage}</wui-text>`:null}};d.styles=[o.W5,s],c([(0,a.MZ)()],d.prototype,"errorMessage",void 0),c([(0,a.MZ)({type:Boolean})],d.prototype,"disabled",void 0),c([(0,a.MZ)()],d.prototype,"value",void 0),c([(0,a.MZ)()],d.prototype,"tabIdx",void 0),d=c([(0,l.E)("wui-email-input")],d)}}]);