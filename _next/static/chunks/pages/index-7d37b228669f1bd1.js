(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[405],{8312:function(e,t,n){(window.__NEXT_P=window.__NEXT_P||[]).push(["/",function(){return n(1863)}])},1863:function(e,t,n){"use strict";n.r(t),n.d(t,{default:function(){return o}});var s=n(5893);let a=e=>{let{suit:t,rank:n}=e,a="flex-1 text-center text-5xl mt-3 ".concat("red-600"==("hearts"===t||"diamonds"===t?"red-600":"black")?"text-red-600":"text-black"," text-opacity-100");return(0,s.jsxs)("div",{className:"flex box-boarder bg-white h-20 w-1/4 p-1 mx-3 px-3 boarder-1 shadow-md",children:[(0,s.jsx)("p",{className:a,children:function(e){switch(e){case"spades":return"♠";case"hearts":return"♥";case"diamonds":return"♦";case"clubs":return"♣"}}(t)}),(0,s.jsx)("p",{className:a,children:function(e){switch(e){case"ace":return"A";case"jack":return"J";case"queen":return"Q";case"king":return"K";default:return e}}(n)})]})},r=e=>{let{player:t}=e;return t?(0,s.jsxs)("div",{className:"flex w-full h-22 p-1 boarder-1 shadow-md bg-slate-50 items-center",children:[(0,s.jsx)("p",{className:"flex-auto text-center text-4xl",children:t.name}),(0,s.jsx)(a,{suit:t.hand[0].suit,rank:t.hand[0].rank}),(0,s.jsx)(a,{suit:t.hand[1].suit,rank:t.hand[1].rank}),(0,s.jsxs)("p",{className:"flex-auto text-center text-4xl",children:[(100*t.equity).toFixed(2),"%"]})]}):(0,s.jsx)("div",{})};var l=n(7294);function c(e){let{hostname:t}=e,[n,a]=(0,l.useState)([]);return((0,l.useEffect)(()=>{if(!t)return;let e=new WebSocket("".concat(t,"/ws"));return e.onmessage=e=>{try{let t=JSON.parse(e.data);a(t.players)}catch(e){console.error("Error parsing JSON:",e)}},e.onerror=e=>{console.error("Websocket error:",e)},()=>{e.close()}},[t]),n)?(0,s.jsx)("div",{className:"grid h-50",children:n.map((e,t)=>(0,s.jsx)(r,{player:e},t))}):(0,s.jsx)("div",{})}function i(e){let{isOpen:t,onClose:n,onSubmit:a}=e,[r,c]=(0,l.useState)(""),i=()=>{a(r),n()};return t?(0,s.jsx)("div",{className:"fixed inset-0 form-control items-center justify-center z-50",children:(0,s.jsxs)("div",{className:"bg-primary-content p-4 rounded",children:[(0,s.jsx)("label",{className:"label",children:(0,s.jsx)("span",{className:"label-text text-xl text-neutral",children:"Endpoint (e.g. wss://192.0.2.1 )"})}),(0,s.jsx)("input",{type:"text",value:r,onChange:e=>c(e.target.value),className:"input input-bordered p-3 text-primary-content"}),(0,s.jsx)("button",{onClick:i,className:"btn btn-primary ml-2",children:"Set"})]})}):null}function o(){let[e,t]=(0,l.useState)(!0),[n,a]=(0,l.useState)("");(0,l.useEffect)(()=>{let e=localStorage.getItem("hostname");return e&&(a(e),t(!1)),()=>{}},[]);let r=e=>{a(e),localStorage.setItem("hostname",e)};return(0,s.jsxs)("main",{className:"flex w-full min-h-screen flex-col items-center justify-between p-2",children:[(0,s.jsx)(i,{isOpen:e,onClose:()=>t(!1),onSubmit:r}),(0,s.jsxs)("div",{className:"flex-1 z-10 w-full max-w-5xl items-center justify-between font-mono text-sm bg-base-100",children:[(0,s.jsxs)("div",{className:"navbar navbar-center bg-base-100 w-full",children:[(0,s.jsx)("a",{className:"btn btn-ghost navbar-start normal-case text-xl text-neutral-50",children:"RFID Poker"}),(0,s.jsx)("div",{className:"navbar-end",children:(0,s.jsx)("button",{onClick:function(){localStorage.removeItem("hostname"),a(""),t(!0)},className:"btn btn-primary normal-case",children:"Remove Endpoint"})})]}),(0,s.jsx)(c,{hostname:n})]})]})}}},function(e){e.O(0,[774,888,179],function(){return e(e.s=8312)}),_N_E=e.O()}]);