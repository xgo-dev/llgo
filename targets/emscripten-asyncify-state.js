// Both link variants share the fiber archive. Select the host implementation
// from the final Emscripten settings.
addToLibrary({
#if ASYNCIFY
  llgo_emscripten_asyncify_state_js__deps: ['$Asyncify'],
#endif
  llgo_emscripten_asyncify_state_js: function() {
#if ASYNCIFY
    return Asyncify.state;
#else
    return 0;
#endif
  },
});
