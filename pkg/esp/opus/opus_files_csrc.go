//go:build llgo && !llgo.idf

// The baremetal (no-IDF) build vendors the whole Opus fixed-point source in
// csrc/ and compiles it with the llgo toolchain: the LLGoFiles constant below
// carries the compile options of the 78/esp-opus CMakeLists (the -D macros,
// fixed-point with the float API off) and every translation unit, one clang
// invocation each. The "$(llvm-config --cflags):" prefix exists to switch
// LLGoFiles into its cflags-carrying form; llvm-config's own output is
// target-neutral (-I and three __STDC_*_MACROS defines) and harmless here.
// No -I is needed: the vendored tree is flat, every quoted include resolves
// inside csrc/. On an IDF build (opus_files_stub.go) the list is empty and
// the project-side glue keeps its role.
package opus

const csrcFiles = `$(llvm-config --cflags) -DHAVE_ALLOCA_H -DHAVE_LRINT -DHAVE_LRINTF -DFIXED_POINT=1 -DDISABLE_FLOAT_API -DHAVE_MEMORY_H -DUSE_ALLOCA -DOPUS_BUILD: csrc/A2NLSF.c; csrc/CNG.c; csrc/HP_variable_cutoff.c; csrc/LPC_analysis_filter.c;
	csrc/LPC_fit.c; csrc/LPC_inv_pred_gain.c; csrc/LP_variable_cutoff.c; csrc/LTP_analysis_filter_FIX.c;
	csrc/LTP_scale_ctrl_FIX.c; csrc/NLSF2A.c; csrc/NLSF_VQ.c; csrc/NLSF_VQ_weights_laroia.c;
	csrc/NLSF_decode.c; csrc/NLSF_del_dec_quant.c; csrc/NLSF_encode.c; csrc/NLSF_stabilize.c;
	csrc/NLSF_unpack.c; csrc/NSQ.c; csrc/NSQ_del_dec.c; csrc/PLC.c;
	csrc/VAD.c; csrc/VQ_WMat_EC.c; csrc/ana_filt_bank_1.c; csrc/apply_sine_window_FIX.c;
	csrc/arm_silk_map.c; csrc/autocorr_FIX.c; csrc/bands.c; csrc/biquad_alt.c;
	csrc/burg_modified_FIX.c; csrc/bwexpander.c; csrc/bwexpander_32.c; csrc/celt.c;
	csrc/celt_decoder.c; csrc/celt_encoder.c; csrc/celt_lpc.c; csrc/check_control_input.c;
	csrc/code_signs.c; csrc/control_SNR.c; csrc/control_audio_bandwidth.c; csrc/control_codec.c;
	csrc/corrMatrix_FIX.c; csrc/cwrs.c; csrc/debug.c; csrc/dec_API.c;
	csrc/decode_core.c; csrc/decode_frame.c; csrc/decode_indices.c; csrc/decode_parameters.c;
	csrc/decode_pitch.c; csrc/decode_pulses.c; csrc/decoder_set_fs.c; csrc/enc_API.c;
	csrc/encode_frame_FIX.c; csrc/encode_indices.c; csrc/encode_pulses.c; csrc/entcode.c;
	csrc/entdec.c; csrc/entenc.c; csrc/extensions.c; csrc/find_LPC_FIX.c;
	csrc/find_LTP_FIX.c; csrc/find_pitch_lags_FIX.c; csrc/find_pred_coefs_FIX.c; csrc/gain_quant.c;
	csrc/init_decoder.c; csrc/init_encoder.c; csrc/inner_prod_aligned.c; csrc/interpolate.c;
	csrc/k2a_FIX.c; csrc/k2a_Q16_FIX.c; csrc/kiss_fft.c; csrc/laplace.c;
	csrc/lin2log.c; csrc/llgo_opus.c; csrc/log2lin.c; csrc/mapping_matrix.c;
	csrc/mathops.c; csrc/mdct.c; csrc/modes.c; csrc/noise_shape_analysis_FIX.c;
	csrc/opus.c; csrc/opus_decoder.c; csrc/opus_encoder.c; csrc/opus_multistream.c;
	csrc/opus_multistream_decoder.c; csrc/opus_multistream_encoder.c; csrc/opus_projection_decoder.c; csrc/opus_projection_encoder.c;
	csrc/pitch.c; csrc/pitch_analysis_core_FIX.c; csrc/pitch_est_tables.c; csrc/process_NLSFs.c;
	csrc/process_gains_FIX.c; csrc/quant_LTP_gains.c; csrc/quant_bands.c; csrc/rate.c;
	csrc/regularize_correlations_FIX.c; csrc/repacketizer.c; csrc/resampler.c; csrc/resampler_down2.c;
	csrc/resampler_down2_3.c; csrc/resampler_private_AR2.c; csrc/resampler_private_IIR_FIR.c; csrc/resampler_private_down_FIR.c;
	csrc/resampler_private_up2_HQ.c; csrc/resampler_rom.c; csrc/residual_energy16_FIX.c; csrc/residual_energy_FIX.c;
	csrc/schur64_FIX.c; csrc/schur_FIX.c; csrc/shell_coder.c; csrc/sigm_Q15.c;
	csrc/sort.c; csrc/stereo_LR_to_MS.c; csrc/stereo_MS_to_LR.c; csrc/stereo_decode_pred.c;
	csrc/stereo_encode_pred.c; csrc/stereo_find_predictor.c; csrc/stereo_quant_pred.c; csrc/sum_sqr_shift.c;
	csrc/table_LSF_cos.c; csrc/tables_LTP.c; csrc/tables_NLSF_CB_NB_MB.c; csrc/tables_NLSF_CB_WB.c;
	csrc/tables_gain.c; csrc/tables_other.c; csrc/tables_pitch_lag.c; csrc/tables_pulses_per_block.c;
	csrc/vector_ops_FIX.c; csrc/vq.c; csrc/warped_autocorrelation_FIX.c;`
