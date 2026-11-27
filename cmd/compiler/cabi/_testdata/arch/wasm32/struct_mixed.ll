; ModuleID = '../../wrap/struct_mixed.c'
source_filename = "../../wrap/struct_mixed.c"
target datalayout = "e-m:e-p:32:32-p10:8:8-p20:8:8-i64:64-f128:64-n32:64-S128-ni:1:10:20"
target triple = "wasm32-unknown-emscripten"

%struct.point1 = type { i8, i16 }
%struct.point2 = type { i8, i32 }
%struct.point3 = type { i8, i64 }
%struct.point4 = type { i8, float }
%struct.point5 = type { i8, double }
%struct.point6 = type { i8, i8* }
%struct.point7 = type { i16, i8 }
%struct.point8 = type { i16, i32 }
%struct.point9 = type { i16, i64 }
%struct.point10 = type { i16, float }
%struct.point11 = type { i16, double }
%struct.point12 = type { i16, i8* }
%struct.point13 = type { i32, i8 }
%struct.point14 = type { i32, i16 }
%struct.point15 = type { i32, i64 }
%struct.point16 = type { i32, float }
%struct.point17 = type { i32, double }
%struct.point18 = type { i32, i8* }
%struct.point19 = type { i64, i8 }
%struct.point20 = type { i64, i16 }
%struct.point21 = type { i64, i32 }
%struct.point22 = type { i64, float }
%struct.point23 = type { i64, double }
%struct.point24 = type { i64, i8* }
%struct.point25 = type { float, i8 }
%struct.point26 = type { float, i16 }
%struct.point27 = type { float, i32 }
%struct.point28 = type { float, i64 }
%struct.point29 = type { float, double }
%struct.point30 = type { float, i8* }
%struct.point31 = type { double, i8 }
%struct.point32 = type { double, i16 }
%struct.point33 = type { double, i32 }
%struct.point34 = type { double, i64 }
%struct.point35 = type { double, float }
%struct.point36 = type { double, i8* }
%struct.point37 = type { i8*, i8 }
%struct.point38 = type { i8*, i16 }
%struct.point39 = type { i8*, i32 }
%struct.point40 = type { i8*, i64 }
%struct.point41 = type { i8*, float }
%struct.point42 = type { i8*, double }
%struct.point43 = type { i8, i16, i8 }
%struct.point44 = type { i8, i16, i8, i8 }
%struct.point45 = type { i8, i16, i8, i8, i8 }
%struct.point46 = type { i8, i16, i8, i8, i16 }
%struct.point47 = type { i8, i16, i8, i8, i32 }
%struct.point48 = type { i8, i16, i8, i8, float }
%struct.point49 = type { i8, i16, i8, i16 }
%struct.point50 = type { i8, i16, i8, i16, i8 }
%struct.point51 = type { i8, i16, i8, i16, i16 }
%struct.point52 = type { i8, i16, i8, i16, i32 }
%struct.point53 = type { i8, i16, i8, i16, float }
%struct.point54 = type { i8, i16, i8, i32 }
%struct.point55 = type { i8, i16, i8, i32, i8 }
%struct.point56 = type { i8, i16, i8, i32, i16 }
%struct.point57 = type { i8, i16, i8, i32, i32 }
%struct.point58 = type { i8, i16, i8, i32, float }
%struct.point59 = type { i8, i16, i8, float }
%struct.point60 = type { i8, i16, i8, float, i8 }
%struct.point61 = type { i8, i16, i8, float, i16 }
%struct.point62 = type { i8, i16, i8, float, i32 }
%struct.point63 = type { i8, i16, i8, float, float }
%struct.point64 = type { i8, i16, i16 }
%struct.point65 = type { i8, i16, i16, i8 }
%struct.point66 = type { i8, i16, i16, i8, i8 }
%struct.point67 = type { i8, i16, i16, i8, i16 }
%struct.point68 = type { i8, i16, i16, i8, i32 }
%struct.point69 = type { i8, i16, i16, i8, float }
%struct.point70 = type { i8, i16, i16, i16 }
%struct.point71 = type { i8, i16, i16, i16, i8 }
%struct.point72 = type { i8, i16, i16, i16, i16 }
%struct.point73 = type { i8, i16, i16, i16, i32 }
%struct.point74 = type { i8, i16, i16, i16, float }
%struct.point75 = type { i8, i16, i16, i32 }
%struct.point76 = type { i8, i16, i16, i32, i8 }
%struct.point77 = type { i8, i16, i16, i32, i16 }
%struct.point78 = type { i8, i16, i16, i32, i32 }
%struct.point79 = type { i8, i16, i16, i32, float }
%struct.point80 = type { i8, i16, i16, float }
%struct.point81 = type { i8, i16, i16, float, i8 }
%struct.point82 = type { i8, i16, i16, float, i16 }
%struct.point83 = type { i8, i16, i16, float, i32 }
%struct.point84 = type { i8, i16, i16, float, float }
%struct.point85 = type { i8, i16, i32 }
%struct.point86 = type { i8, i16, i32, i8 }
%struct.point87 = type { i8, i16, i32, i8, i8 }
%struct.point88 = type { i8, i16, i32, i8, i16 }
%struct.point89 = type { i8, i16, i32, i8, i32 }
%struct.point90 = type { i8, i16, i32, i8, float }
%struct.point91 = type { i8, i16, i32, i16 }
%struct.point92 = type { i8, i16, i32, i16, i8 }
%struct.point93 = type { i8, i16, i32, i16, i16 }
%struct.point94 = type { i8, i16, i32, i16, i32 }
%struct.point95 = type { i8, i16, i32, i16, float }
%struct.point96 = type { i8, i16, i32, i32 }
%struct.point97 = type { i8, i16, i32, i32, i8 }
%struct.point98 = type { i8, i16, i32, i32, i16 }
%struct.point99 = type { i8, i16, i32, i32, i32 }
%struct.point100 = type { i8, i16, i32, i32, float }
%struct.point101 = type { i8, i16, i32, float }
%struct.point102 = type { i8, i16, i32, float, i8 }
%struct.point103 = type { i8, i16, i32, float, i16 }
%struct.point104 = type { i8, i16, i32, float, i32 }
%struct.point105 = type { i8, i16, i32, float, float }
%struct.point106 = type { i8, i16, float }
%struct.point107 = type { i8, i16, float, i8 }
%struct.point108 = type { i8, i16, float, i8, i8 }
%struct.point109 = type { i8, i16, float, i8, i16 }
%struct.point110 = type { i8, i16, float, i8, i32 }
%struct.point111 = type { i8, i16, float, i8, float }
%struct.point112 = type { i8, i16, float, i16 }
%struct.point113 = type { i8, i16, float, i16, i8 }
%struct.point114 = type { i8, i16, float, i16, i16 }
%struct.point115 = type { i8, i16, float, i16, i32 }
%struct.point116 = type { i8, i16, float, i16, float }
%struct.point117 = type { i8, i16, float, i32 }
%struct.point118 = type { i8, i16, float, i32, i8 }
%struct.point119 = type { i8, i16, float, i32, i16 }
%struct.point120 = type { i8, i16, float, i32, i32 }
%struct.point121 = type { i8, i16, float, i32, float }
%struct.point122 = type { i8, i16, float, float }
%struct.point123 = type { i8, i16, float, float, i8 }
%struct.point124 = type { i8, i16, float, float, i16 }
%struct.point125 = type { i8, i16, float, float, i32 }
%struct.point126 = type { i8, i16, float, float, float }
%struct.point127 = type { i8, i32, i8 }
%struct.point128 = type { i8, i32, i8, i8 }
%struct.point129 = type { i8, i32, i8, i8, i8 }
%struct.point130 = type { i8, i32, i8, i8, i16 }
%struct.point131 = type { i8, i32, i8, i8, i32 }
%struct.point132 = type { i8, i32, i8, i8, float }
%struct.point133 = type { i8, i32, i8, i16 }
%struct.point134 = type { i8, i32, i8, i16, i8 }
%struct.point135 = type { i8, i32, i8, i16, i16 }
%struct.point136 = type { i8, i32, i8, i16, i32 }
%struct.point137 = type { i8, i32, i8, i16, float }
%struct.point138 = type { i8, i32, i8, i32 }
%struct.point139 = type { i8, i32, i8, i32, i8 }
%struct.point140 = type { i8, i32, i8, i32, i16 }
%struct.point141 = type { i8, i32, i8, i32, i32 }
%struct.point142 = type { i8, i32, i8, i32, float }
%struct.point143 = type { i8, i32, i8, float }
%struct.point144 = type { i8, i32, i8, float, i8 }
%struct.point145 = type { i8, i32, i8, float, i16 }
%struct.point146 = type { i8, i32, i8, float, i32 }
%struct.point147 = type { i8, i32, i8, float, float }
%struct.point148 = type { i8, i32, i16 }
%struct.point149 = type { i8, i32, i16, i8 }
%struct.point150 = type { i8, i32, i16, i8, i8 }
%struct.point151 = type { i8, i32, i16, i8, i16 }
%struct.point152 = type { i8, i32, i16, i8, i32 }
%struct.point153 = type { i8, i32, i16, i8, float }
%struct.point154 = type { i8, i32, i16, i16 }
%struct.point155 = type { i8, i32, i16, i16, i8 }
%struct.point156 = type { i8, i32, i16, i16, i16 }
%struct.point157 = type { i8, i32, i16, i16, i32 }
%struct.point158 = type { i8, i32, i16, i16, float }
%struct.point159 = type { i8, i32, i16, i32 }
%struct.point160 = type { i8, i32, i16, i32, i8 }
%struct.point161 = type { i8, i32, i16, i32, i16 }
%struct.point162 = type { i8, i32, i16, i32, i32 }
%struct.point163 = type { i8, i32, i16, i32, float }
%struct.point164 = type { i8, i32, i16, float }
%struct.point165 = type { i8, i32, i16, float, i8 }
%struct.point166 = type { i8, i32, i16, float, i16 }
%struct.point167 = type { i8, i32, i16, float, i32 }
%struct.point168 = type { i8, i32, i16, float, float }
%struct.point169 = type { i8, i32, i32 }
%struct.point170 = type { i8, i32, i32, i8 }
%struct.point171 = type { i8, i32, i32, i8, i8 }
%struct.point172 = type { i8, i32, i32, i8, i16 }
%struct.point173 = type { i8, i32, i32, i8, i32 }
%struct.point174 = type { i8, i32, i32, i8, float }
%struct.point175 = type { i8, i32, i32, i16 }
%struct.point176 = type { i8, i32, i32, i16, i8 }
%struct.point177 = type { i8, i32, i32, i16, i16 }
%struct.point178 = type { i8, i32, i32, i16, i32 }
%struct.point179 = type { i8, i32, i32, i16, float }
%struct.point180 = type { i8, i32, i32, i32 }
%struct.point181 = type { i8, i32, i32, i32, i8 }
%struct.point182 = type { i8, i32, i32, i32, i16 }
%struct.point183 = type { i8, i32, i32, i32, i32 }
%struct.point184 = type { i8, i32, i32, i32, float }
%struct.point185 = type { i8, i32, i32, float }
%struct.point186 = type { i8, i32, i32, float, i8 }
%struct.point187 = type { i8, i32, i32, float, i16 }
%struct.point188 = type { i8, i32, i32, float, i32 }
%struct.point189 = type { i8, i32, i32, float, float }
%struct.point190 = type { i8, i32, float }
%struct.point191 = type { i8, i32, float, i8 }
%struct.point192 = type { i8, i32, float, i8, i8 }
%struct.point193 = type { i8, i32, float, i8, i16 }
%struct.point194 = type { i8, i32, float, i8, i32 }
%struct.point195 = type { i8, i32, float, i8, float }
%struct.point196 = type { i8, i32, float, i16 }
%struct.point197 = type { i8, i32, float, i16, i8 }
%struct.point198 = type { i8, i32, float, i16, i16 }
%struct.point199 = type { i8, i32, float, i16, i32 }
%struct.point200 = type { i8, i32, float, i16, float }
%struct.point201 = type { i8, i32, float, i32 }
%struct.point202 = type { i8, i32, float, i32, i8 }
%struct.point203 = type { i8, i32, float, i32, i16 }
%struct.point204 = type { i8, i32, float, i32, i32 }
%struct.point205 = type { i8, i32, float, i32, float }
%struct.point206 = type { i8, i32, float, float }
%struct.point207 = type { i8, i32, float, float, i8 }
%struct.point208 = type { i8, i32, float, float, i16 }
%struct.point209 = type { i8, i32, float, float, i32 }
%struct.point210 = type { i8, i32, float, float, float }
%struct.point211 = type { i8, float, i8 }
%struct.point212 = type { i8, float, i8, i8 }
%struct.point213 = type { i8, float, i8, i8, i8 }
%struct.point214 = type { i8, float, i8, i8, i16 }
%struct.point215 = type { i8, float, i8, i8, i32 }
%struct.point216 = type { i8, float, i8, i8, float }
%struct.point217 = type { i8, float, i8, i16 }
%struct.point218 = type { i8, float, i8, i16, i8 }
%struct.point219 = type { i8, float, i8, i16, i16 }
%struct.point220 = type { i8, float, i8, i16, i32 }
%struct.point221 = type { i8, float, i8, i16, float }
%struct.point222 = type { i8, float, i8, i32 }
%struct.point223 = type { i8, float, i8, i32, i8 }
%struct.point224 = type { i8, float, i8, i32, i16 }
%struct.point225 = type { i8, float, i8, i32, i32 }
%struct.point226 = type { i8, float, i8, i32, float }
%struct.point227 = type { i8, float, i8, float }
%struct.point228 = type { i8, float, i8, float, i8 }
%struct.point229 = type { i8, float, i8, float, i16 }
%struct.point230 = type { i8, float, i8, float, i32 }
%struct.point231 = type { i8, float, i8, float, float }
%struct.point232 = type { i8, float, i16 }
%struct.point233 = type { i8, float, i16, i8 }
%struct.point234 = type { i8, float, i16, i8, i8 }
%struct.point235 = type { i8, float, i16, i8, i16 }
%struct.point236 = type { i8, float, i16, i8, i32 }
%struct.point237 = type { i8, float, i16, i8, float }
%struct.point238 = type { i8, float, i16, i16 }
%struct.point239 = type { i8, float, i16, i16, i8 }
%struct.point240 = type { i8, float, i16, i16, i16 }
%struct.point241 = type { i8, float, i16, i16, i32 }
%struct.point242 = type { i8, float, i16, i16, float }
%struct.point243 = type { i8, float, i16, i32 }
%struct.point244 = type { i8, float, i16, i32, i8 }
%struct.point245 = type { i8, float, i16, i32, i16 }
%struct.point246 = type { i8, float, i16, i32, i32 }
%struct.point247 = type { i8, float, i16, i32, float }
%struct.point248 = type { i8, float, i16, float }
%struct.point249 = type { i8, float, i16, float, i8 }
%struct.point250 = type { i8, float, i16, float, i16 }
%struct.point251 = type { i8, float, i16, float, i32 }
%struct.point252 = type { i8, float, i16, float, float }
%struct.point253 = type { i8, float, i32 }
%struct.point254 = type { i8, float, i32, i8 }
%struct.point255 = type { i8, float, i32, i8, i8 }
%struct.point256 = type { i8, float, i32, i8, i16 }
%struct.point257 = type { i8, float, i32, i8, i32 }
%struct.point258 = type { i8, float, i32, i8, float }
%struct.point259 = type { i8, float, i32, i16 }
%struct.point260 = type { i8, float, i32, i16, i8 }
%struct.point261 = type { i8, float, i32, i16, i16 }
%struct.point262 = type { i8, float, i32, i16, i32 }
%struct.point263 = type { i8, float, i32, i16, float }
%struct.point264 = type { i8, float, i32, i32 }
%struct.point265 = type { i8, float, i32, i32, i8 }
%struct.point266 = type { i8, float, i32, i32, i16 }
%struct.point267 = type { i8, float, i32, i32, i32 }
%struct.point268 = type { i8, float, i32, i32, float }
%struct.point269 = type { i8, float, i32, float }
%struct.point270 = type { i8, float, i32, float, i8 }
%struct.point271 = type { i8, float, i32, float, i16 }
%struct.point272 = type { i8, float, i32, float, i32 }
%struct.point273 = type { i8, float, i32, float, float }
%struct.point274 = type { i8, float, float }
%struct.point275 = type { i8, float, float, i8 }
%struct.point276 = type { i8, float, float, i8, i8 }
%struct.point277 = type { i8, float, float, i8, i16 }
%struct.point278 = type { i8, float, float, i8, i32 }
%struct.point279 = type { i8, float, float, i8, float }
%struct.point280 = type { i8, float, float, i16 }
%struct.point281 = type { i8, float, float, i16, i8 }
%struct.point282 = type { i8, float, float, i16, i16 }
%struct.point283 = type { i8, float, float, i16, i32 }
%struct.point284 = type { i8, float, float, i16, float }
%struct.point285 = type { i8, float, float, i32 }
%struct.point286 = type { i8, float, float, i32, i8 }
%struct.point287 = type { i8, float, float, i32, i16 }
%struct.point288 = type { i8, float, float, i32, i32 }
%struct.point289 = type { i8, float, float, i32, float }
%struct.point290 = type { i8, float, float, float }
%struct.point291 = type { i8, float, float, float, i8 }
%struct.point292 = type { i8, float, float, float, i16 }
%struct.point293 = type { i8, float, float, float, i32 }
%struct.point294 = type { i8, float, float, float, float }
%struct.point295 = type { i16, i8, i8 }
%struct.point296 = type { i16, i8, i8, i8 }
%struct.point297 = type { i16, i8, i8, i8, i8 }
%struct.point298 = type { i16, i8, i8, i8, i16 }
%struct.point299 = type { i16, i8, i8, i8, i32 }
%struct.point300 = type { i16, i8, i8, i8, float }
%struct.point301 = type { i16, i8, i8, i16 }
%struct.point302 = type { i16, i8, i8, i16, i8 }
%struct.point303 = type { i16, i8, i8, i16, i16 }
%struct.point304 = type { i16, i8, i8, i16, i32 }
%struct.point305 = type { i16, i8, i8, i16, float }
%struct.point306 = type { i16, i8, i8, i32 }
%struct.point307 = type { i16, i8, i8, i32, i8 }
%struct.point308 = type { i16, i8, i8, i32, i16 }
%struct.point309 = type { i16, i8, i8, i32, i32 }
%struct.point310 = type { i16, i8, i8, i32, float }
%struct.point311 = type { i16, i8, i8, float }
%struct.point312 = type { i16, i8, i8, float, i8 }
%struct.point313 = type { i16, i8, i8, float, i16 }
%struct.point314 = type { i16, i8, i8, float, i32 }
%struct.point315 = type { i16, i8, i8, float, float }
%struct.point316 = type { i16, i8, i16 }
%struct.point317 = type { i16, i8, i16, i8 }
%struct.point318 = type { i16, i8, i16, i8, i8 }
%struct.point319 = type { i16, i8, i16, i8, i16 }
%struct.point320 = type { i16, i8, i16, i8, i32 }
%struct.point321 = type { i16, i8, i16, i8, float }
%struct.point322 = type { i16, i8, i16, i16 }
%struct.point323 = type { i16, i8, i16, i16, i8 }
%struct.point324 = type { i16, i8, i16, i16, i16 }
%struct.point325 = type { i16, i8, i16, i16, i32 }
%struct.point326 = type { i16, i8, i16, i16, float }
%struct.point327 = type { i16, i8, i16, i32 }
%struct.point328 = type { i16, i8, i16, i32, i8 }
%struct.point329 = type { i16, i8, i16, i32, i16 }
%struct.point330 = type { i16, i8, i16, i32, i32 }
%struct.point331 = type { i16, i8, i16, i32, float }
%struct.point332 = type { i16, i8, i16, float }
%struct.point333 = type { i16, i8, i16, float, i8 }
%struct.point334 = type { i16, i8, i16, float, i16 }
%struct.point335 = type { i16, i8, i16, float, i32 }
%struct.point336 = type { i16, i8, i16, float, float }
%struct.point337 = type { i16, i8, i32 }
%struct.point338 = type { i16, i8, i32, i8 }
%struct.point339 = type { i16, i8, i32, i8, i8 }
%struct.point340 = type { i16, i8, i32, i8, i16 }
%struct.point341 = type { i16, i8, i32, i8, i32 }
%struct.point342 = type { i16, i8, i32, i8, float }
%struct.point343 = type { i16, i8, i32, i16 }
%struct.point344 = type { i16, i8, i32, i16, i8 }
%struct.point345 = type { i16, i8, i32, i16, i16 }
%struct.point346 = type { i16, i8, i32, i16, i32 }
%struct.point347 = type { i16, i8, i32, i16, float }
%struct.point348 = type { i16, i8, i32, i32 }
%struct.point349 = type { i16, i8, i32, i32, i8 }
%struct.point350 = type { i16, i8, i32, i32, i16 }
%struct.point351 = type { i16, i8, i32, i32, i32 }
%struct.point352 = type { i16, i8, i32, i32, float }
%struct.point353 = type { i16, i8, i32, float }
%struct.point354 = type { i16, i8, i32, float, i8 }
%struct.point355 = type { i16, i8, i32, float, i16 }
%struct.point356 = type { i16, i8, i32, float, i32 }
%struct.point357 = type { i16, i8, i32, float, float }
%struct.point358 = type { i16, i8, float }
%struct.point359 = type { i16, i8, float, i8 }
%struct.point360 = type { i16, i8, float, i8, i8 }
%struct.point361 = type { i16, i8, float, i8, i16 }
%struct.point362 = type { i16, i8, float, i8, i32 }
%struct.point363 = type { i16, i8, float, i8, float }
%struct.point364 = type { i16, i8, float, i16 }
%struct.point365 = type { i16, i8, float, i16, i8 }
%struct.point366 = type { i16, i8, float, i16, i16 }
%struct.point367 = type { i16, i8, float, i16, i32 }
%struct.point368 = type { i16, i8, float, i16, float }
%struct.point369 = type { i16, i8, float, i32 }
%struct.point370 = type { i16, i8, float, i32, i8 }
%struct.point371 = type { i16, i8, float, i32, i16 }
%struct.point372 = type { i16, i8, float, i32, i32 }
%struct.point373 = type { i16, i8, float, i32, float }
%struct.point374 = type { i16, i8, float, float }
%struct.point375 = type { i16, i8, float, float, i8 }
%struct.point376 = type { i16, i8, float, float, i16 }
%struct.point377 = type { i16, i8, float, float, i32 }
%struct.point378 = type { i16, i8, float, float, float }
%struct.point379 = type { i16, i32, i8 }
%struct.point380 = type { i16, i32, i8, i8 }
%struct.point381 = type { i16, i32, i8, i8, i8 }
%struct.point382 = type { i16, i32, i8, i8, i16 }
%struct.point383 = type { i16, i32, i8, i8, i32 }
%struct.point384 = type { i16, i32, i8, i8, float }
%struct.point385 = type { i16, i32, i8, i16 }
%struct.point386 = type { i16, i32, i8, i16, i8 }
%struct.point387 = type { i16, i32, i8, i16, i16 }
%struct.point388 = type { i16, i32, i8, i16, i32 }
%struct.point389 = type { i16, i32, i8, i16, float }
%struct.point390 = type { i16, i32, i8, i32 }
%struct.point391 = type { i16, i32, i8, i32, i8 }
%struct.point392 = type { i16, i32, i8, i32, i16 }
%struct.point393 = type { i16, i32, i8, i32, i32 }
%struct.point394 = type { i16, i32, i8, i32, float }
%struct.point395 = type { i16, i32, i8, float }
%struct.point396 = type { i16, i32, i8, float, i8 }
%struct.point397 = type { i16, i32, i8, float, i16 }
%struct.point398 = type { i16, i32, i8, float, i32 }
%struct.point399 = type { i16, i32, i8, float, float }
%struct.point400 = type { i16, i32, i16 }
%struct.point401 = type { i16, i32, i16, i8 }
%struct.point402 = type { i16, i32, i16, i8, i8 }
%struct.point403 = type { i16, i32, i16, i8, i16 }
%struct.point404 = type { i16, i32, i16, i8, i32 }
%struct.point405 = type { i16, i32, i16, i8, float }
%struct.point406 = type { i16, i32, i16, i16 }
%struct.point407 = type { i16, i32, i16, i16, i8 }
%struct.point408 = type { i16, i32, i16, i16, i16 }
%struct.point409 = type { i16, i32, i16, i16, i32 }
%struct.point410 = type { i16, i32, i16, i16, float }
%struct.point411 = type { i16, i32, i16, i32 }
%struct.point412 = type { i16, i32, i16, i32, i8 }
%struct.point413 = type { i16, i32, i16, i32, i16 }
%struct.point414 = type { i16, i32, i16, i32, i32 }
%struct.point415 = type { i16, i32, i16, i32, float }
%struct.point416 = type { i16, i32, i16, float }
%struct.point417 = type { i16, i32, i16, float, i8 }
%struct.point418 = type { i16, i32, i16, float, i16 }
%struct.point419 = type { i16, i32, i16, float, i32 }
%struct.point420 = type { i16, i32, i16, float, float }
%struct.point421 = type { i16, i32, i32 }
%struct.point422 = type { i16, i32, i32, i8 }
%struct.point423 = type { i16, i32, i32, i8, i8 }
%struct.point424 = type { i16, i32, i32, i8, i16 }
%struct.point425 = type { i16, i32, i32, i8, i32 }
%struct.point426 = type { i16, i32, i32, i8, float }
%struct.point427 = type { i16, i32, i32, i16 }
%struct.point428 = type { i16, i32, i32, i16, i8 }
%struct.point429 = type { i16, i32, i32, i16, i16 }
%struct.point430 = type { i16, i32, i32, i16, i32 }
%struct.point431 = type { i16, i32, i32, i16, float }
%struct.point432 = type { i16, i32, i32, i32 }
%struct.point433 = type { i16, i32, i32, i32, i8 }
%struct.point434 = type { i16, i32, i32, i32, i16 }
%struct.point435 = type { i16, i32, i32, i32, i32 }
%struct.point436 = type { i16, i32, i32, i32, float }
%struct.point437 = type { i16, i32, i32, float }
%struct.point438 = type { i16, i32, i32, float, i8 }
%struct.point439 = type { i16, i32, i32, float, i16 }
%struct.point440 = type { i16, i32, i32, float, i32 }
%struct.point441 = type { i16, i32, i32, float, float }
%struct.point442 = type { i16, i32, float }
%struct.point443 = type { i16, i32, float, i8 }
%struct.point444 = type { i16, i32, float, i8, i8 }
%struct.point445 = type { i16, i32, float, i8, i16 }
%struct.point446 = type { i16, i32, float, i8, i32 }
%struct.point447 = type { i16, i32, float, i8, float }
%struct.point448 = type { i16, i32, float, i16 }
%struct.point449 = type { i16, i32, float, i16, i8 }
%struct.point450 = type { i16, i32, float, i16, i16 }
%struct.point451 = type { i16, i32, float, i16, i32 }
%struct.point452 = type { i16, i32, float, i16, float }
%struct.point453 = type { i16, i32, float, i32 }
%struct.point454 = type { i16, i32, float, i32, i8 }
%struct.point455 = type { i16, i32, float, i32, i16 }
%struct.point456 = type { i16, i32, float, i32, i32 }
%struct.point457 = type { i16, i32, float, i32, float }
%struct.point458 = type { i16, i32, float, float }
%struct.point459 = type { i16, i32, float, float, i8 }
%struct.point460 = type { i16, i32, float, float, i16 }
%struct.point461 = type { i16, i32, float, float, i32 }
%struct.point462 = type { i16, i32, float, float, float }
%struct.point463 = type { i16, float, i8 }
%struct.point464 = type { i16, float, i8, i8 }
%struct.point465 = type { i16, float, i8, i8, i8 }
%struct.point466 = type { i16, float, i8, i8, i16 }
%struct.point467 = type { i16, float, i8, i8, i32 }
%struct.point468 = type { i16, float, i8, i8, float }
%struct.point469 = type { i16, float, i8, i16 }
%struct.point470 = type { i16, float, i8, i16, i8 }
%struct.point471 = type { i16, float, i8, i16, i16 }
%struct.point472 = type { i16, float, i8, i16, i32 }
%struct.point473 = type { i16, float, i8, i16, float }
%struct.point474 = type { i16, float, i8, i32 }
%struct.point475 = type { i16, float, i8, i32, i8 }
%struct.point476 = type { i16, float, i8, i32, i16 }
%struct.point477 = type { i16, float, i8, i32, i32 }
%struct.point478 = type { i16, float, i8, i32, float }
%struct.point479 = type { i16, float, i8, float }
%struct.point480 = type { i16, float, i8, float, i8 }
%struct.point481 = type { i16, float, i8, float, i16 }
%struct.point482 = type { i16, float, i8, float, i32 }
%struct.point483 = type { i16, float, i8, float, float }
%struct.point484 = type { i16, float, i16 }
%struct.point485 = type { i16, float, i16, i8 }
%struct.point486 = type { i16, float, i16, i8, i8 }
%struct.point487 = type { i16, float, i16, i8, i16 }
%struct.point488 = type { i16, float, i16, i8, i32 }
%struct.point489 = type { i16, float, i16, i8, float }
%struct.point490 = type { i16, float, i16, i16 }
%struct.point491 = type { i16, float, i16, i16, i8 }
%struct.point492 = type { i16, float, i16, i16, i16 }
%struct.point493 = type { i16, float, i16, i16, i32 }
%struct.point494 = type { i16, float, i16, i16, float }
%struct.point495 = type { i16, float, i16, i32 }
%struct.point496 = type { i16, float, i16, i32, i8 }
%struct.point497 = type { i16, float, i16, i32, i16 }
%struct.point498 = type { i16, float, i16, i32, i32 }
%struct.point499 = type { i16, float, i16, i32, float }
%struct.point500 = type { i16, float, i16, float }
%struct.point501 = type { i16, float, i16, float, i8 }
%struct.point502 = type { i16, float, i16, float, i16 }
%struct.point503 = type { i16, float, i16, float, i32 }
%struct.point504 = type { i16, float, i16, float, float }
%struct.point505 = type { i16, float, i32 }
%struct.point506 = type { i16, float, i32, i8 }
%struct.point507 = type { i16, float, i32, i8, i8 }
%struct.point508 = type { i16, float, i32, i8, i16 }
%struct.point509 = type { i16, float, i32, i8, i32 }
%struct.point510 = type { i16, float, i32, i8, float }
%struct.point511 = type { i16, float, i32, i16 }
%struct.point512 = type { i16, float, i32, i16, i8 }
%struct.point513 = type { i16, float, i32, i16, i16 }
%struct.point514 = type { i16, float, i32, i16, i32 }
%struct.point515 = type { i16, float, i32, i16, float }
%struct.point516 = type { i16, float, i32, i32 }
%struct.point517 = type { i16, float, i32, i32, i8 }
%struct.point518 = type { i16, float, i32, i32, i16 }
%struct.point519 = type { i16, float, i32, i32, i32 }
%struct.point520 = type { i16, float, i32, i32, float }
%struct.point521 = type { i16, float, i32, float }
%struct.point522 = type { i16, float, i32, float, i8 }
%struct.point523 = type { i16, float, i32, float, i16 }
%struct.point524 = type { i16, float, i32, float, i32 }
%struct.point525 = type { i16, float, i32, float, float }
%struct.point526 = type { i16, float, float }
%struct.point527 = type { i16, float, float, i8 }
%struct.point528 = type { i16, float, float, i8, i8 }
%struct.point529 = type { i16, float, float, i8, i16 }
%struct.point530 = type { i16, float, float, i8, i32 }
%struct.point531 = type { i16, float, float, i8, float }
%struct.point532 = type { i16, float, float, i16 }
%struct.point533 = type { i16, float, float, i16, i8 }
%struct.point534 = type { i16, float, float, i16, i16 }
%struct.point535 = type { i16, float, float, i16, i32 }
%struct.point536 = type { i16, float, float, i16, float }
%struct.point537 = type { i16, float, float, i32 }
%struct.point538 = type { i16, float, float, i32, i8 }
%struct.point539 = type { i16, float, float, i32, i16 }
%struct.point540 = type { i16, float, float, i32, i32 }
%struct.point541 = type { i16, float, float, i32, float }
%struct.point542 = type { i16, float, float, float }
%struct.point543 = type { i16, float, float, float, i8 }
%struct.point544 = type { i16, float, float, float, i16 }
%struct.point545 = type { i16, float, float, float, i32 }
%struct.point546 = type { i16, float, float, float, float }
%struct.point547 = type { i32, i8, i8 }
%struct.point548 = type { i32, i8, i8, i8 }
%struct.point549 = type { i32, i8, i8, i8, i8 }
%struct.point550 = type { i32, i8, i8, i8, i16 }
%struct.point551 = type { i32, i8, i8, i8, i32 }
%struct.point552 = type { i32, i8, i8, i8, float }
%struct.point553 = type { i32, i8, i8, i16 }
%struct.point554 = type { i32, i8, i8, i16, i8 }
%struct.point555 = type { i32, i8, i8, i16, i16 }
%struct.point556 = type { i32, i8, i8, i16, i32 }
%struct.point557 = type { i32, i8, i8, i16, float }
%struct.point558 = type { i32, i8, i8, i32 }
%struct.point559 = type { i32, i8, i8, i32, i8 }
%struct.point560 = type { i32, i8, i8, i32, i16 }
%struct.point561 = type { i32, i8, i8, i32, i32 }
%struct.point562 = type { i32, i8, i8, i32, float }
%struct.point563 = type { i32, i8, i8, float }
%struct.point564 = type { i32, i8, i8, float, i8 }
%struct.point565 = type { i32, i8, i8, float, i16 }
%struct.point566 = type { i32, i8, i8, float, i32 }
%struct.point567 = type { i32, i8, i8, float, float }
%struct.point568 = type { i32, i8, i16 }
%struct.point569 = type { i32, i8, i16, i8 }
%struct.point570 = type { i32, i8, i16, i8, i8 }
%struct.point571 = type { i32, i8, i16, i8, i16 }
%struct.point572 = type { i32, i8, i16, i8, i32 }
%struct.point573 = type { i32, i8, i16, i8, float }
%struct.point574 = type { i32, i8, i16, i16 }
%struct.point575 = type { i32, i8, i16, i16, i8 }
%struct.point576 = type { i32, i8, i16, i16, i16 }
%struct.point577 = type { i32, i8, i16, i16, i32 }
%struct.point578 = type { i32, i8, i16, i16, float }
%struct.point579 = type { i32, i8, i16, i32 }
%struct.point580 = type { i32, i8, i16, i32, i8 }
%struct.point581 = type { i32, i8, i16, i32, i16 }
%struct.point582 = type { i32, i8, i16, i32, i32 }
%struct.point583 = type { i32, i8, i16, i32, float }
%struct.point584 = type { i32, i8, i16, float }
%struct.point585 = type { i32, i8, i16, float, i8 }
%struct.point586 = type { i32, i8, i16, float, i16 }
%struct.point587 = type { i32, i8, i16, float, i32 }
%struct.point588 = type { i32, i8, i16, float, float }
%struct.point589 = type { i32, i8, i32 }
%struct.point590 = type { i32, i8, i32, i8 }
%struct.point591 = type { i32, i8, i32, i8, i8 }
%struct.point592 = type { i32, i8, i32, i8, i16 }
%struct.point593 = type { i32, i8, i32, i8, i32 }
%struct.point594 = type { i32, i8, i32, i8, float }
%struct.point595 = type { i32, i8, i32, i16 }
%struct.point596 = type { i32, i8, i32, i16, i8 }
%struct.point597 = type { i32, i8, i32, i16, i16 }
%struct.point598 = type { i32, i8, i32, i16, i32 }
%struct.point599 = type { i32, i8, i32, i16, float }
%struct.point600 = type { i32, i8, i32, i32 }
%struct.point601 = type { i32, i8, i32, i32, i8 }
%struct.point602 = type { i32, i8, i32, i32, i16 }
%struct.point603 = type { i32, i8, i32, i32, i32 }
%struct.point604 = type { i32, i8, i32, i32, float }
%struct.point605 = type { i32, i8, i32, float }
%struct.point606 = type { i32, i8, i32, float, i8 }
%struct.point607 = type { i32, i8, i32, float, i16 }
%struct.point608 = type { i32, i8, i32, float, i32 }
%struct.point609 = type { i32, i8, i32, float, float }
%struct.point610 = type { i32, i8, float }
%struct.point611 = type { i32, i8, float, i8 }
%struct.point612 = type { i32, i8, float, i8, i8 }
%struct.point613 = type { i32, i8, float, i8, i16 }
%struct.point614 = type { i32, i8, float, i8, i32 }
%struct.point615 = type { i32, i8, float, i8, float }
%struct.point616 = type { i32, i8, float, i16 }
%struct.point617 = type { i32, i8, float, i16, i8 }
%struct.point618 = type { i32, i8, float, i16, i16 }
%struct.point619 = type { i32, i8, float, i16, i32 }
%struct.point620 = type { i32, i8, float, i16, float }
%struct.point621 = type { i32, i8, float, i32 }
%struct.point622 = type { i32, i8, float, i32, i8 }
%struct.point623 = type { i32, i8, float, i32, i16 }
%struct.point624 = type { i32, i8, float, i32, i32 }
%struct.point625 = type { i32, i8, float, i32, float }
%struct.point626 = type { i32, i8, float, float }
%struct.point627 = type { i32, i8, float, float, i8 }
%struct.point628 = type { i32, i8, float, float, i16 }
%struct.point629 = type { i32, i8, float, float, i32 }
%struct.point630 = type { i32, i8, float, float, float }
%struct.point631 = type { i32, i16, i8 }
%struct.point632 = type { i32, i16, i8, i8 }
%struct.point633 = type { i32, i16, i8, i8, i8 }
%struct.point634 = type { i32, i16, i8, i8, i16 }
%struct.point635 = type { i32, i16, i8, i8, i32 }
%struct.point636 = type { i32, i16, i8, i8, float }
%struct.point637 = type { i32, i16, i8, i16 }
%struct.point638 = type { i32, i16, i8, i16, i8 }
%struct.point639 = type { i32, i16, i8, i16, i16 }
%struct.point640 = type { i32, i16, i8, i16, i32 }
%struct.point641 = type { i32, i16, i8, i16, float }
%struct.point642 = type { i32, i16, i8, i32 }
%struct.point643 = type { i32, i16, i8, i32, i8 }
%struct.point644 = type { i32, i16, i8, i32, i16 }
%struct.point645 = type { i32, i16, i8, i32, i32 }
%struct.point646 = type { i32, i16, i8, i32, float }
%struct.point647 = type { i32, i16, i8, float }
%struct.point648 = type { i32, i16, i8, float, i8 }
%struct.point649 = type { i32, i16, i8, float, i16 }
%struct.point650 = type { i32, i16, i8, float, i32 }
%struct.point651 = type { i32, i16, i8, float, float }
%struct.point652 = type { i32, i16, i16 }
%struct.point653 = type { i32, i16, i16, i8 }
%struct.point654 = type { i32, i16, i16, i8, i8 }
%struct.point655 = type { i32, i16, i16, i8, i16 }
%struct.point656 = type { i32, i16, i16, i8, i32 }
%struct.point657 = type { i32, i16, i16, i8, float }
%struct.point658 = type { i32, i16, i16, i16 }
%struct.point659 = type { i32, i16, i16, i16, i8 }
%struct.point660 = type { i32, i16, i16, i16, i16 }
%struct.point661 = type { i32, i16, i16, i16, i32 }
%struct.point662 = type { i32, i16, i16, i16, float }
%struct.point663 = type { i32, i16, i16, i32 }
%struct.point664 = type { i32, i16, i16, i32, i8 }
%struct.point665 = type { i32, i16, i16, i32, i16 }
%struct.point666 = type { i32, i16, i16, i32, i32 }
%struct.point667 = type { i32, i16, i16, i32, float }
%struct.point668 = type { i32, i16, i16, float }
%struct.point669 = type { i32, i16, i16, float, i8 }
%struct.point670 = type { i32, i16, i16, float, i16 }
%struct.point671 = type { i32, i16, i16, float, i32 }
%struct.point672 = type { i32, i16, i16, float, float }
%struct.point673 = type { i32, i16, i32 }
%struct.point674 = type { i32, i16, i32, i8 }
%struct.point675 = type { i32, i16, i32, i8, i8 }
%struct.point676 = type { i32, i16, i32, i8, i16 }
%struct.point677 = type { i32, i16, i32, i8, i32 }
%struct.point678 = type { i32, i16, i32, i8, float }
%struct.point679 = type { i32, i16, i32, i16 }
%struct.point680 = type { i32, i16, i32, i16, i8 }
%struct.point681 = type { i32, i16, i32, i16, i16 }
%struct.point682 = type { i32, i16, i32, i16, i32 }
%struct.point683 = type { i32, i16, i32, i16, float }
%struct.point684 = type { i32, i16, i32, i32 }
%struct.point685 = type { i32, i16, i32, i32, i8 }
%struct.point686 = type { i32, i16, i32, i32, i16 }
%struct.point687 = type { i32, i16, i32, i32, i32 }
%struct.point688 = type { i32, i16, i32, i32, float }
%struct.point689 = type { i32, i16, i32, float }
%struct.point690 = type { i32, i16, i32, float, i8 }
%struct.point691 = type { i32, i16, i32, float, i16 }
%struct.point692 = type { i32, i16, i32, float, i32 }
%struct.point693 = type { i32, i16, i32, float, float }
%struct.point694 = type { i32, i16, float }
%struct.point695 = type { i32, i16, float, i8 }
%struct.point696 = type { i32, i16, float, i8, i8 }
%struct.point697 = type { i32, i16, float, i8, i16 }
%struct.point698 = type { i32, i16, float, i8, i32 }
%struct.point699 = type { i32, i16, float, i8, float }
%struct.point700 = type { i32, i16, float, i16 }
%struct.point701 = type { i32, i16, float, i16, i8 }
%struct.point702 = type { i32, i16, float, i16, i16 }
%struct.point703 = type { i32, i16, float, i16, i32 }
%struct.point704 = type { i32, i16, float, i16, float }
%struct.point705 = type { i32, i16, float, i32 }
%struct.point706 = type { i32, i16, float, i32, i8 }
%struct.point707 = type { i32, i16, float, i32, i16 }
%struct.point708 = type { i32, i16, float, i32, i32 }
%struct.point709 = type { i32, i16, float, i32, float }
%struct.point710 = type { i32, i16, float, float }
%struct.point711 = type { i32, i16, float, float, i8 }
%struct.point712 = type { i32, i16, float, float, i16 }
%struct.point713 = type { i32, i16, float, float, i32 }
%struct.point714 = type { i32, i16, float, float, float }
%struct.point715 = type { i32, float, i8 }
%struct.point716 = type { i32, float, i8, i8 }
%struct.point717 = type { i32, float, i8, i8, i8 }
%struct.point718 = type { i32, float, i8, i8, i16 }
%struct.point719 = type { i32, float, i8, i8, i32 }
%struct.point720 = type { i32, float, i8, i8, float }
%struct.point721 = type { i32, float, i8, i16 }
%struct.point722 = type { i32, float, i8, i16, i8 }
%struct.point723 = type { i32, float, i8, i16, i16 }
%struct.point724 = type { i32, float, i8, i16, i32 }
%struct.point725 = type { i32, float, i8, i16, float }
%struct.point726 = type { i32, float, i8, i32 }
%struct.point727 = type { i32, float, i8, i32, i8 }
%struct.point728 = type { i32, float, i8, i32, i16 }
%struct.point729 = type { i32, float, i8, i32, i32 }
%struct.point730 = type { i32, float, i8, i32, float }
%struct.point731 = type { i32, float, i8, float }
%struct.point732 = type { i32, float, i8, float, i8 }
%struct.point733 = type { i32, float, i8, float, i16 }
%struct.point734 = type { i32, float, i8, float, i32 }
%struct.point735 = type { i32, float, i8, float, float }
%struct.point736 = type { i32, float, i16 }
%struct.point737 = type { i32, float, i16, i8 }
%struct.point738 = type { i32, float, i16, i8, i8 }
%struct.point739 = type { i32, float, i16, i8, i16 }
%struct.point740 = type { i32, float, i16, i8, i32 }
%struct.point741 = type { i32, float, i16, i8, float }
%struct.point742 = type { i32, float, i16, i16 }
%struct.point743 = type { i32, float, i16, i16, i8 }
%struct.point744 = type { i32, float, i16, i16, i16 }
%struct.point745 = type { i32, float, i16, i16, i32 }
%struct.point746 = type { i32, float, i16, i16, float }
%struct.point747 = type { i32, float, i16, i32 }
%struct.point748 = type { i32, float, i16, i32, i8 }
%struct.point749 = type { i32, float, i16, i32, i16 }
%struct.point750 = type { i32, float, i16, i32, i32 }
%struct.point751 = type { i32, float, i16, i32, float }
%struct.point752 = type { i32, float, i16, float }
%struct.point753 = type { i32, float, i16, float, i8 }
%struct.point754 = type { i32, float, i16, float, i16 }
%struct.point755 = type { i32, float, i16, float, i32 }
%struct.point756 = type { i32, float, i16, float, float }
%struct.point757 = type { i32, float, i32 }
%struct.point758 = type { i32, float, i32, i8 }
%struct.point759 = type { i32, float, i32, i8, i8 }
%struct.point760 = type { i32, float, i32, i8, i16 }
%struct.point761 = type { i32, float, i32, i8, i32 }
%struct.point762 = type { i32, float, i32, i8, float }
%struct.point763 = type { i32, float, i32, i16 }
%struct.point764 = type { i32, float, i32, i16, i8 }
%struct.point765 = type { i32, float, i32, i16, i16 }
%struct.point766 = type { i32, float, i32, i16, i32 }
%struct.point767 = type { i32, float, i32, i16, float }
%struct.point768 = type { i32, float, i32, i32 }
%struct.point769 = type { i32, float, i32, i32, i8 }
%struct.point770 = type { i32, float, i32, i32, i16 }
%struct.point771 = type { i32, float, i32, i32, i32 }
%struct.point772 = type { i32, float, i32, i32, float }
%struct.point773 = type { i32, float, i32, float }
%struct.point774 = type { i32, float, i32, float, i8 }
%struct.point775 = type { i32, float, i32, float, i16 }
%struct.point776 = type { i32, float, i32, float, i32 }
%struct.point777 = type { i32, float, i32, float, float }
%struct.point778 = type { i32, float, float }
%struct.point779 = type { i32, float, float, i8 }
%struct.point780 = type { i32, float, float, i8, i8 }
%struct.point781 = type { i32, float, float, i8, i16 }
%struct.point782 = type { i32, float, float, i8, i32 }
%struct.point783 = type { i32, float, float, i8, float }
%struct.point784 = type { i32, float, float, i16 }
%struct.point785 = type { i32, float, float, i16, i8 }
%struct.point786 = type { i32, float, float, i16, i16 }
%struct.point787 = type { i32, float, float, i16, i32 }
%struct.point788 = type { i32, float, float, i16, float }
%struct.point789 = type { i32, float, float, i32 }
%struct.point790 = type { i32, float, float, i32, i8 }
%struct.point791 = type { i32, float, float, i32, i16 }
%struct.point792 = type { i32, float, float, i32, i32 }
%struct.point793 = type { i32, float, float, i32, float }
%struct.point794 = type { i32, float, float, float }
%struct.point795 = type { i32, float, float, float, i8 }
%struct.point796 = type { i32, float, float, float, i16 }
%struct.point797 = type { i32, float, float, float, i32 }
%struct.point798 = type { i32, float, float, float, float }
%struct.point799 = type { float, i8, i8 }
%struct.point800 = type { float, i8, i8, i8 }
%struct.point801 = type { float, i8, i8, i8, i8 }
%struct.point802 = type { float, i8, i8, i8, i16 }
%struct.point803 = type { float, i8, i8, i8, i32 }
%struct.point804 = type { float, i8, i8, i8, float }
%struct.point805 = type { float, i8, i8, i16 }
%struct.point806 = type { float, i8, i8, i16, i8 }
%struct.point807 = type { float, i8, i8, i16, i16 }
%struct.point808 = type { float, i8, i8, i16, i32 }
%struct.point809 = type { float, i8, i8, i16, float }
%struct.point810 = type { float, i8, i8, i32 }
%struct.point811 = type { float, i8, i8, i32, i8 }
%struct.point812 = type { float, i8, i8, i32, i16 }
%struct.point813 = type { float, i8, i8, i32, i32 }
%struct.point814 = type { float, i8, i8, i32, float }
%struct.point815 = type { float, i8, i8, float }
%struct.point816 = type { float, i8, i8, float, i8 }
%struct.point817 = type { float, i8, i8, float, i16 }
%struct.point818 = type { float, i8, i8, float, i32 }
%struct.point819 = type { float, i8, i8, float, float }
%struct.point820 = type { float, i8, i16 }
%struct.point821 = type { float, i8, i16, i8 }
%struct.point822 = type { float, i8, i16, i8, i8 }
%struct.point823 = type { float, i8, i16, i8, i16 }
%struct.point824 = type { float, i8, i16, i8, i32 }
%struct.point825 = type { float, i8, i16, i8, float }
%struct.point826 = type { float, i8, i16, i16 }
%struct.point827 = type { float, i8, i16, i16, i8 }
%struct.point828 = type { float, i8, i16, i16, i16 }
%struct.point829 = type { float, i8, i16, i16, i32 }
%struct.point830 = type { float, i8, i16, i16, float }
%struct.point831 = type { float, i8, i16, i32 }
%struct.point832 = type { float, i8, i16, i32, i8 }
%struct.point833 = type { float, i8, i16, i32, i16 }
%struct.point834 = type { float, i8, i16, i32, i32 }
%struct.point835 = type { float, i8, i16, i32, float }
%struct.point836 = type { float, i8, i16, float }
%struct.point837 = type { float, i8, i16, float, i8 }
%struct.point838 = type { float, i8, i16, float, i16 }
%struct.point839 = type { float, i8, i16, float, i32 }
%struct.point840 = type { float, i8, i16, float, float }
%struct.point841 = type { float, i8, i32 }
%struct.point842 = type { float, i8, i32, i8 }
%struct.point843 = type { float, i8, i32, i8, i8 }
%struct.point844 = type { float, i8, i32, i8, i16 }
%struct.point845 = type { float, i8, i32, i8, i32 }
%struct.point846 = type { float, i8, i32, i8, float }
%struct.point847 = type { float, i8, i32, i16 }
%struct.point848 = type { float, i8, i32, i16, i8 }
%struct.point849 = type { float, i8, i32, i16, i16 }
%struct.point850 = type { float, i8, i32, i16, i32 }
%struct.point851 = type { float, i8, i32, i16, float }
%struct.point852 = type { float, i8, i32, i32 }
%struct.point853 = type { float, i8, i32, i32, i8 }
%struct.point854 = type { float, i8, i32, i32, i16 }
%struct.point855 = type { float, i8, i32, i32, i32 }
%struct.point856 = type { float, i8, i32, i32, float }
%struct.point857 = type { float, i8, i32, float }
%struct.point858 = type { float, i8, i32, float, i8 }
%struct.point859 = type { float, i8, i32, float, i16 }
%struct.point860 = type { float, i8, i32, float, i32 }
%struct.point861 = type { float, i8, i32, float, float }
%struct.point862 = type { float, i8, float }
%struct.point863 = type { float, i8, float, i8 }
%struct.point864 = type { float, i8, float, i8, i8 }
%struct.point865 = type { float, i8, float, i8, i16 }
%struct.point866 = type { float, i8, float, i8, i32 }
%struct.point867 = type { float, i8, float, i8, float }
%struct.point868 = type { float, i8, float, i16 }
%struct.point869 = type { float, i8, float, i16, i8 }
%struct.point870 = type { float, i8, float, i16, i16 }
%struct.point871 = type { float, i8, float, i16, i32 }
%struct.point872 = type { float, i8, float, i16, float }
%struct.point873 = type { float, i8, float, i32 }
%struct.point874 = type { float, i8, float, i32, i8 }
%struct.point875 = type { float, i8, float, i32, i16 }
%struct.point876 = type { float, i8, float, i32, i32 }
%struct.point877 = type { float, i8, float, i32, float }
%struct.point878 = type { float, i8, float, float }
%struct.point879 = type { float, i8, float, float, i8 }
%struct.point880 = type { float, i8, float, float, i16 }
%struct.point881 = type { float, i8, float, float, i32 }
%struct.point882 = type { float, i8, float, float, float }
%struct.point883 = type { float, i16, i8 }
%struct.point884 = type { float, i16, i8, i8 }
%struct.point885 = type { float, i16, i8, i8, i8 }
%struct.point886 = type { float, i16, i8, i8, i16 }
%struct.point887 = type { float, i16, i8, i8, i32 }
%struct.point888 = type { float, i16, i8, i8, float }
%struct.point889 = type { float, i16, i8, i16 }
%struct.point890 = type { float, i16, i8, i16, i8 }
%struct.point891 = type { float, i16, i8, i16, i16 }
%struct.point892 = type { float, i16, i8, i16, i32 }
%struct.point893 = type { float, i16, i8, i16, float }
%struct.point894 = type { float, i16, i8, i32 }
%struct.point895 = type { float, i16, i8, i32, i8 }
%struct.point896 = type { float, i16, i8, i32, i16 }
%struct.point897 = type { float, i16, i8, i32, i32 }
%struct.point898 = type { float, i16, i8, i32, float }
%struct.point899 = type { float, i16, i8, float }
%struct.point900 = type { float, i16, i8, float, i8 }
%struct.point901 = type { float, i16, i8, float, i16 }
%struct.point902 = type { float, i16, i8, float, i32 }
%struct.point903 = type { float, i16, i8, float, float }
%struct.point904 = type { float, i16, i16 }
%struct.point905 = type { float, i16, i16, i8 }
%struct.point906 = type { float, i16, i16, i8, i8 }
%struct.point907 = type { float, i16, i16, i8, i16 }
%struct.point908 = type { float, i16, i16, i8, i32 }
%struct.point909 = type { float, i16, i16, i8, float }
%struct.point910 = type { float, i16, i16, i16 }
%struct.point911 = type { float, i16, i16, i16, i8 }
%struct.point912 = type { float, i16, i16, i16, i16 }
%struct.point913 = type { float, i16, i16, i16, i32 }
%struct.point914 = type { float, i16, i16, i16, float }
%struct.point915 = type { float, i16, i16, i32 }
%struct.point916 = type { float, i16, i16, i32, i8 }
%struct.point917 = type { float, i16, i16, i32, i16 }
%struct.point918 = type { float, i16, i16, i32, i32 }
%struct.point919 = type { float, i16, i16, i32, float }
%struct.point920 = type { float, i16, i16, float }
%struct.point921 = type { float, i16, i16, float, i8 }
%struct.point922 = type { float, i16, i16, float, i16 }
%struct.point923 = type { float, i16, i16, float, i32 }
%struct.point924 = type { float, i16, i16, float, float }
%struct.point925 = type { float, i16, i32 }
%struct.point926 = type { float, i16, i32, i8 }
%struct.point927 = type { float, i16, i32, i8, i8 }
%struct.point928 = type { float, i16, i32, i8, i16 }
%struct.point929 = type { float, i16, i32, i8, i32 }
%struct.point930 = type { float, i16, i32, i8, float }
%struct.point931 = type { float, i16, i32, i16 }
%struct.point932 = type { float, i16, i32, i16, i8 }
%struct.point933 = type { float, i16, i32, i16, i16 }
%struct.point934 = type { float, i16, i32, i16, i32 }
%struct.point935 = type { float, i16, i32, i16, float }
%struct.point936 = type { float, i16, i32, i32 }
%struct.point937 = type { float, i16, i32, i32, i8 }
%struct.point938 = type { float, i16, i32, i32, i16 }
%struct.point939 = type { float, i16, i32, i32, i32 }
%struct.point940 = type { float, i16, i32, i32, float }
%struct.point941 = type { float, i16, i32, float }
%struct.point942 = type { float, i16, i32, float, i8 }
%struct.point943 = type { float, i16, i32, float, i16 }
%struct.point944 = type { float, i16, i32, float, i32 }
%struct.point945 = type { float, i16, i32, float, float }
%struct.point946 = type { float, i16, float }
%struct.point947 = type { float, i16, float, i8 }
%struct.point948 = type { float, i16, float, i8, i8 }
%struct.point949 = type { float, i16, float, i8, i16 }
%struct.point950 = type { float, i16, float, i8, i32 }
%struct.point951 = type { float, i16, float, i8, float }
%struct.point952 = type { float, i16, float, i16 }
%struct.point953 = type { float, i16, float, i16, i8 }
%struct.point954 = type { float, i16, float, i16, i16 }
%struct.point955 = type { float, i16, float, i16, i32 }
%struct.point956 = type { float, i16, float, i16, float }
%struct.point957 = type { float, i16, float, i32 }
%struct.point958 = type { float, i16, float, i32, i8 }
%struct.point959 = type { float, i16, float, i32, i16 }
%struct.point960 = type { float, i16, float, i32, i32 }
%struct.point961 = type { float, i16, float, i32, float }
%struct.point962 = type { float, i16, float, float }
%struct.point963 = type { float, i16, float, float, i8 }
%struct.point964 = type { float, i16, float, float, i16 }
%struct.point965 = type { float, i16, float, float, i32 }
%struct.point966 = type { float, i16, float, float, float }
%struct.point967 = type { float, i32, i8 }
%struct.point968 = type { float, i32, i8, i8 }
%struct.point969 = type { float, i32, i8, i8, i8 }
%struct.point970 = type { float, i32, i8, i8, i16 }
%struct.point971 = type { float, i32, i8, i8, i32 }
%struct.point972 = type { float, i32, i8, i8, float }
%struct.point973 = type { float, i32, i8, i16 }
%struct.point974 = type { float, i32, i8, i16, i8 }
%struct.point975 = type { float, i32, i8, i16, i16 }
%struct.point976 = type { float, i32, i8, i16, i32 }
%struct.point977 = type { float, i32, i8, i16, float }
%struct.point978 = type { float, i32, i8, i32 }
%struct.point979 = type { float, i32, i8, i32, i8 }
%struct.point980 = type { float, i32, i8, i32, i16 }
%struct.point981 = type { float, i32, i8, i32, i32 }
%struct.point982 = type { float, i32, i8, i32, float }
%struct.point983 = type { float, i32, i8, float }
%struct.point984 = type { float, i32, i8, float, i8 }
%struct.point985 = type { float, i32, i8, float, i16 }
%struct.point986 = type { float, i32, i8, float, i32 }
%struct.point987 = type { float, i32, i8, float, float }
%struct.point988 = type { float, i32, i16 }
%struct.point989 = type { float, i32, i16, i8 }
%struct.point990 = type { float, i32, i16, i8, i8 }
%struct.point991 = type { float, i32, i16, i8, i16 }
%struct.point992 = type { float, i32, i16, i8, i32 }
%struct.point993 = type { float, i32, i16, i8, float }
%struct.point994 = type { float, i32, i16, i16 }
%struct.point995 = type { float, i32, i16, i16, i8 }
%struct.point996 = type { float, i32, i16, i16, i16 }
%struct.point997 = type { float, i32, i16, i16, i32 }
%struct.point998 = type { float, i32, i16, i16, float }
%struct.point999 = type { float, i32, i16, i32 }
%struct.point1000 = type { float, i32, i16, i32, i8 }
%struct.point1001 = type { float, i32, i16, i32, i16 }
%struct.point1002 = type { float, i32, i16, i32, i32 }
%struct.point1003 = type { float, i32, i16, i32, float }
%struct.point1004 = type { float, i32, i16, float }
%struct.point1005 = type { float, i32, i16, float, i8 }
%struct.point1006 = type { float, i32, i16, float, i16 }
%struct.point1007 = type { float, i32, i16, float, i32 }
%struct.point1008 = type { float, i32, i16, float, float }
%struct.point1009 = type { float, i32, i32 }
%struct.point1010 = type { float, i32, i32, i8 }
%struct.point1011 = type { float, i32, i32, i8, i8 }
%struct.point1012 = type { float, i32, i32, i8, i16 }
%struct.point1013 = type { float, i32, i32, i8, i32 }
%struct.point1014 = type { float, i32, i32, i8, float }
%struct.point1015 = type { float, i32, i32, i16 }
%struct.point1016 = type { float, i32, i32, i16, i8 }
%struct.point1017 = type { float, i32, i32, i16, i16 }
%struct.point1018 = type { float, i32, i32, i16, i32 }
%struct.point1019 = type { float, i32, i32, i16, float }
%struct.point1020 = type { float, i32, i32, i32 }
%struct.point1021 = type { float, i32, i32, i32, i8 }
%struct.point1022 = type { float, i32, i32, i32, i16 }
%struct.point1023 = type { float, i32, i32, i32, i32 }
%struct.point1024 = type { float, i32, i32, i32, float }
%struct.point1025 = type { float, i32, i32, float }
%struct.point1026 = type { float, i32, i32, float, i8 }
%struct.point1027 = type { float, i32, i32, float, i16 }
%struct.point1028 = type { float, i32, i32, float, i32 }
%struct.point1029 = type { float, i32, i32, float, float }
%struct.point1030 = type { float, i32, float }
%struct.point1031 = type { float, i32, float, i8 }
%struct.point1032 = type { float, i32, float, i8, i8 }
%struct.point1033 = type { float, i32, float, i8, i16 }
%struct.point1034 = type { float, i32, float, i8, i32 }
%struct.point1035 = type { float, i32, float, i8, float }
%struct.point1036 = type { float, i32, float, i16 }
%struct.point1037 = type { float, i32, float, i16, i8 }
%struct.point1038 = type { float, i32, float, i16, i16 }
%struct.point1039 = type { float, i32, float, i16, i32 }
%struct.point1040 = type { float, i32, float, i16, float }
%struct.point1041 = type { float, i32, float, i32 }
%struct.point1042 = type { float, i32, float, i32, i8 }
%struct.point1043 = type { float, i32, float, i32, i16 }
%struct.point1044 = type { float, i32, float, i32, i32 }
%struct.point1045 = type { float, i32, float, i32, float }
%struct.point1046 = type { float, i32, float, float }
%struct.point1047 = type { float, i32, float, float, i8 }
%struct.point1048 = type { float, i32, float, float, i16 }
%struct.point1049 = type { float, i32, float, float, i32 }
%struct.point1050 = type { float, i32, float, float, float }

; Function Attrs: noinline nounwind optnone
define hidden void @demo1(%struct.point1* noalias sret(%struct.point1) align 2 %0, %struct.point1* noundef byval(%struct.point1) align 2 %1) #0 {
  %3 = bitcast %struct.point1* %0 to i8*
  %4 = bitcast %struct.point1* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 4, i1 false)
  ret void
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i32(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define hidden void @demo2(%struct.point2* noalias sret(%struct.point2) align 4 %0, %struct.point2* noundef byval(%struct.point2) align 4 %1) #0 {
  %3 = bitcast %struct.point2* %0 to i8*
  %4 = bitcast %struct.point2* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo3(%struct.point3* noalias sret(%struct.point3) align 8 %0, %struct.point3* noundef byval(%struct.point3) align 8 %1) #0 {
  %3 = bitcast %struct.point3* %0 to i8*
  %4 = bitcast %struct.point3* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo4(%struct.point4* noalias sret(%struct.point4) align 4 %0, %struct.point4* noundef byval(%struct.point4) align 4 %1) #0 {
  %3 = bitcast %struct.point4* %0 to i8*
  %4 = bitcast %struct.point4* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo5(%struct.point5* noalias sret(%struct.point5) align 8 %0, %struct.point5* noundef byval(%struct.point5) align 8 %1) #0 {
  %3 = bitcast %struct.point5* %0 to i8*
  %4 = bitcast %struct.point5* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo6(%struct.point6* noalias sret(%struct.point6) align 4 %0, %struct.point6* noundef byval(%struct.point6) align 4 %1) #0 {
  %3 = bitcast %struct.point6* %0 to i8*
  %4 = bitcast %struct.point6* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo7(%struct.point7* noalias sret(%struct.point7) align 2 %0, %struct.point7* noundef byval(%struct.point7) align 2 %1) #0 {
  %3 = bitcast %struct.point7* %0 to i8*
  %4 = bitcast %struct.point7* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 4, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo8(%struct.point8* noalias sret(%struct.point8) align 4 %0, %struct.point8* noundef byval(%struct.point8) align 4 %1) #0 {
  %3 = bitcast %struct.point8* %0 to i8*
  %4 = bitcast %struct.point8* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo9(%struct.point9* noalias sret(%struct.point9) align 8 %0, %struct.point9* noundef byval(%struct.point9) align 8 %1) #0 {
  %3 = bitcast %struct.point9* %0 to i8*
  %4 = bitcast %struct.point9* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo10(%struct.point10* noalias sret(%struct.point10) align 4 %0, %struct.point10* noundef byval(%struct.point10) align 4 %1) #0 {
  %3 = bitcast %struct.point10* %0 to i8*
  %4 = bitcast %struct.point10* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo11(%struct.point11* noalias sret(%struct.point11) align 8 %0, %struct.point11* noundef byval(%struct.point11) align 8 %1) #0 {
  %3 = bitcast %struct.point11* %0 to i8*
  %4 = bitcast %struct.point11* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo12(%struct.point12* noalias sret(%struct.point12) align 4 %0, %struct.point12* noundef byval(%struct.point12) align 4 %1) #0 {
  %3 = bitcast %struct.point12* %0 to i8*
  %4 = bitcast %struct.point12* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo13(%struct.point13* noalias sret(%struct.point13) align 4 %0, %struct.point13* noundef byval(%struct.point13) align 4 %1) #0 {
  %3 = bitcast %struct.point13* %0 to i8*
  %4 = bitcast %struct.point13* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo14(%struct.point14* noalias sret(%struct.point14) align 4 %0, %struct.point14* noundef byval(%struct.point14) align 4 %1) #0 {
  %3 = bitcast %struct.point14* %0 to i8*
  %4 = bitcast %struct.point14* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo15(%struct.point15* noalias sret(%struct.point15) align 8 %0, %struct.point15* noundef byval(%struct.point15) align 8 %1) #0 {
  %3 = bitcast %struct.point15* %0 to i8*
  %4 = bitcast %struct.point15* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo16(%struct.point16* noalias sret(%struct.point16) align 4 %0, %struct.point16* noundef byval(%struct.point16) align 4 %1) #0 {
  %3 = bitcast %struct.point16* %0 to i8*
  %4 = bitcast %struct.point16* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo17(%struct.point17* noalias sret(%struct.point17) align 8 %0, %struct.point17* noundef byval(%struct.point17) align 8 %1) #0 {
  %3 = bitcast %struct.point17* %0 to i8*
  %4 = bitcast %struct.point17* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo18(%struct.point18* noalias sret(%struct.point18) align 4 %0, %struct.point18* noundef byval(%struct.point18) align 4 %1) #0 {
  %3 = bitcast %struct.point18* %0 to i8*
  %4 = bitcast %struct.point18* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo19(%struct.point19* noalias sret(%struct.point19) align 8 %0, %struct.point19* noundef byval(%struct.point19) align 8 %1) #0 {
  %3 = bitcast %struct.point19* %0 to i8*
  %4 = bitcast %struct.point19* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo20(%struct.point20* noalias sret(%struct.point20) align 8 %0, %struct.point20* noundef byval(%struct.point20) align 8 %1) #0 {
  %3 = bitcast %struct.point20* %0 to i8*
  %4 = bitcast %struct.point20* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo21(%struct.point21* noalias sret(%struct.point21) align 8 %0, %struct.point21* noundef byval(%struct.point21) align 8 %1) #0 {
  %3 = bitcast %struct.point21* %0 to i8*
  %4 = bitcast %struct.point21* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo22(%struct.point22* noalias sret(%struct.point22) align 8 %0, %struct.point22* noundef byval(%struct.point22) align 8 %1) #0 {
  %3 = bitcast %struct.point22* %0 to i8*
  %4 = bitcast %struct.point22* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo23(%struct.point23* noalias sret(%struct.point23) align 8 %0, %struct.point23* noundef byval(%struct.point23) align 8 %1) #0 {
  %3 = bitcast %struct.point23* %0 to i8*
  %4 = bitcast %struct.point23* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo24(%struct.point24* noalias sret(%struct.point24) align 8 %0, %struct.point24* noundef byval(%struct.point24) align 8 %1) #0 {
  %3 = bitcast %struct.point24* %0 to i8*
  %4 = bitcast %struct.point24* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo25(%struct.point25* noalias sret(%struct.point25) align 4 %0, %struct.point25* noundef byval(%struct.point25) align 4 %1) #0 {
  %3 = bitcast %struct.point25* %0 to i8*
  %4 = bitcast %struct.point25* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo26(%struct.point26* noalias sret(%struct.point26) align 4 %0, %struct.point26* noundef byval(%struct.point26) align 4 %1) #0 {
  %3 = bitcast %struct.point26* %0 to i8*
  %4 = bitcast %struct.point26* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo27(%struct.point27* noalias sret(%struct.point27) align 4 %0, %struct.point27* noundef byval(%struct.point27) align 4 %1) #0 {
  %3 = bitcast %struct.point27* %0 to i8*
  %4 = bitcast %struct.point27* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo28(%struct.point28* noalias sret(%struct.point28) align 8 %0, %struct.point28* noundef byval(%struct.point28) align 8 %1) #0 {
  %3 = bitcast %struct.point28* %0 to i8*
  %4 = bitcast %struct.point28* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo29(%struct.point29* noalias sret(%struct.point29) align 8 %0, %struct.point29* noundef byval(%struct.point29) align 8 %1) #0 {
  %3 = bitcast %struct.point29* %0 to i8*
  %4 = bitcast %struct.point29* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo30(%struct.point30* noalias sret(%struct.point30) align 4 %0, %struct.point30* noundef byval(%struct.point30) align 4 %1) #0 {
  %3 = bitcast %struct.point30* %0 to i8*
  %4 = bitcast %struct.point30* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo31(%struct.point31* noalias sret(%struct.point31) align 8 %0, %struct.point31* noundef byval(%struct.point31) align 8 %1) #0 {
  %3 = bitcast %struct.point31* %0 to i8*
  %4 = bitcast %struct.point31* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo32(%struct.point32* noalias sret(%struct.point32) align 8 %0, %struct.point32* noundef byval(%struct.point32) align 8 %1) #0 {
  %3 = bitcast %struct.point32* %0 to i8*
  %4 = bitcast %struct.point32* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo33(%struct.point33* noalias sret(%struct.point33) align 8 %0, %struct.point33* noundef byval(%struct.point33) align 8 %1) #0 {
  %3 = bitcast %struct.point33* %0 to i8*
  %4 = bitcast %struct.point33* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo34(%struct.point34* noalias sret(%struct.point34) align 8 %0, %struct.point34* noundef byval(%struct.point34) align 8 %1) #0 {
  %3 = bitcast %struct.point34* %0 to i8*
  %4 = bitcast %struct.point34* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo35(%struct.point35* noalias sret(%struct.point35) align 8 %0, %struct.point35* noundef byval(%struct.point35) align 8 %1) #0 {
  %3 = bitcast %struct.point35* %0 to i8*
  %4 = bitcast %struct.point35* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo36(%struct.point36* noalias sret(%struct.point36) align 8 %0, %struct.point36* noundef byval(%struct.point36) align 8 %1) #0 {
  %3 = bitcast %struct.point36* %0 to i8*
  %4 = bitcast %struct.point36* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo37(%struct.point37* noalias sret(%struct.point37) align 4 %0, %struct.point37* noundef byval(%struct.point37) align 4 %1) #0 {
  %3 = bitcast %struct.point37* %0 to i8*
  %4 = bitcast %struct.point37* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo38(%struct.point38* noalias sret(%struct.point38) align 4 %0, %struct.point38* noundef byval(%struct.point38) align 4 %1) #0 {
  %3 = bitcast %struct.point38* %0 to i8*
  %4 = bitcast %struct.point38* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo39(%struct.point39* noalias sret(%struct.point39) align 4 %0, %struct.point39* noundef byval(%struct.point39) align 4 %1) #0 {
  %3 = bitcast %struct.point39* %0 to i8*
  %4 = bitcast %struct.point39* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo40(%struct.point40* noalias sret(%struct.point40) align 8 %0, %struct.point40* noundef byval(%struct.point40) align 8 %1) #0 {
  %3 = bitcast %struct.point40* %0 to i8*
  %4 = bitcast %struct.point40* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo41(%struct.point41* noalias sret(%struct.point41) align 4 %0, %struct.point41* noundef byval(%struct.point41) align 4 %1) #0 {
  %3 = bitcast %struct.point41* %0 to i8*
  %4 = bitcast %struct.point41* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo42(%struct.point42* noalias sret(%struct.point42) align 8 %0, %struct.point42* noundef byval(%struct.point42) align 8 %1) #0 {
  %3 = bitcast %struct.point42* %0 to i8*
  %4 = bitcast %struct.point42* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 8 %3, i8* align 8 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo43(%struct.point43* noalias sret(%struct.point43) align 2 %0, %struct.point43* noundef byval(%struct.point43) align 2 %1) #0 {
  %3 = bitcast %struct.point43* %0 to i8*
  %4 = bitcast %struct.point43* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 6, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo44(%struct.point44* noalias sret(%struct.point44) align 2 %0, %struct.point44* noundef byval(%struct.point44) align 2 %1) #0 {
  %3 = bitcast %struct.point44* %0 to i8*
  %4 = bitcast %struct.point44* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 6, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo45(%struct.point45* noalias sret(%struct.point45) align 2 %0, %struct.point45* noundef byval(%struct.point45) align 2 %1) #0 {
  %3 = bitcast %struct.point45* %0 to i8*
  %4 = bitcast %struct.point45* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo46(%struct.point46* noalias sret(%struct.point46) align 2 %0, %struct.point46* noundef byval(%struct.point46) align 2 %1) #0 {
  %3 = bitcast %struct.point46* %0 to i8*
  %4 = bitcast %struct.point46* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo47(%struct.point47* noalias sret(%struct.point47) align 4 %0, %struct.point47* noundef byval(%struct.point47) align 4 %1) #0 {
  %3 = bitcast %struct.point47* %0 to i8*
  %4 = bitcast %struct.point47* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo48(%struct.point48* noalias sret(%struct.point48) align 4 %0, %struct.point48* noundef byval(%struct.point48) align 4 %1) #0 {
  %3 = bitcast %struct.point48* %0 to i8*
  %4 = bitcast %struct.point48* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo49(%struct.point49* noalias sret(%struct.point49) align 2 %0, %struct.point49* noundef byval(%struct.point49) align 2 %1) #0 {
  %3 = bitcast %struct.point49* %0 to i8*
  %4 = bitcast %struct.point49* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo50(%struct.point50* noalias sret(%struct.point50) align 2 %0, %struct.point50* noundef byval(%struct.point50) align 2 %1) #0 {
  %3 = bitcast %struct.point50* %0 to i8*
  %4 = bitcast %struct.point50* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo51(%struct.point51* noalias sret(%struct.point51) align 2 %0, %struct.point51* noundef byval(%struct.point51) align 2 %1) #0 {
  %3 = bitcast %struct.point51* %0 to i8*
  %4 = bitcast %struct.point51* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo52(%struct.point52* noalias sret(%struct.point52) align 4 %0, %struct.point52* noundef byval(%struct.point52) align 4 %1) #0 {
  %3 = bitcast %struct.point52* %0 to i8*
  %4 = bitcast %struct.point52* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo53(%struct.point53* noalias sret(%struct.point53) align 4 %0, %struct.point53* noundef byval(%struct.point53) align 4 %1) #0 {
  %3 = bitcast %struct.point53* %0 to i8*
  %4 = bitcast %struct.point53* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo54(%struct.point54* noalias sret(%struct.point54) align 4 %0, %struct.point54* noundef byval(%struct.point54) align 4 %1) #0 {
  %3 = bitcast %struct.point54* %0 to i8*
  %4 = bitcast %struct.point54* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo55(%struct.point55* noalias sret(%struct.point55) align 4 %0, %struct.point55* noundef byval(%struct.point55) align 4 %1) #0 {
  %3 = bitcast %struct.point55* %0 to i8*
  %4 = bitcast %struct.point55* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo56(%struct.point56* noalias sret(%struct.point56) align 4 %0, %struct.point56* noundef byval(%struct.point56) align 4 %1) #0 {
  %3 = bitcast %struct.point56* %0 to i8*
  %4 = bitcast %struct.point56* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo57(%struct.point57* noalias sret(%struct.point57) align 4 %0, %struct.point57* noundef byval(%struct.point57) align 4 %1) #0 {
  %3 = bitcast %struct.point57* %0 to i8*
  %4 = bitcast %struct.point57* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo58(%struct.point58* noalias sret(%struct.point58) align 4 %0, %struct.point58* noundef byval(%struct.point58) align 4 %1) #0 {
  %3 = bitcast %struct.point58* %0 to i8*
  %4 = bitcast %struct.point58* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo59(%struct.point59* noalias sret(%struct.point59) align 4 %0, %struct.point59* noundef byval(%struct.point59) align 4 %1) #0 {
  %3 = bitcast %struct.point59* %0 to i8*
  %4 = bitcast %struct.point59* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo60(%struct.point60* noalias sret(%struct.point60) align 4 %0, %struct.point60* noundef byval(%struct.point60) align 4 %1) #0 {
  %3 = bitcast %struct.point60* %0 to i8*
  %4 = bitcast %struct.point60* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo61(%struct.point61* noalias sret(%struct.point61) align 4 %0, %struct.point61* noundef byval(%struct.point61) align 4 %1) #0 {
  %3 = bitcast %struct.point61* %0 to i8*
  %4 = bitcast %struct.point61* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo62(%struct.point62* noalias sret(%struct.point62) align 4 %0, %struct.point62* noundef byval(%struct.point62) align 4 %1) #0 {
  %3 = bitcast %struct.point62* %0 to i8*
  %4 = bitcast %struct.point62* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo63(%struct.point63* noalias sret(%struct.point63) align 4 %0, %struct.point63* noundef byval(%struct.point63) align 4 %1) #0 {
  %3 = bitcast %struct.point63* %0 to i8*
  %4 = bitcast %struct.point63* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo64(%struct.point64* noalias sret(%struct.point64) align 2 %0, %struct.point64* noundef byval(%struct.point64) align 2 %1) #0 {
  %3 = bitcast %struct.point64* %0 to i8*
  %4 = bitcast %struct.point64* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 6, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo65(%struct.point65* noalias sret(%struct.point65) align 2 %0, %struct.point65* noundef byval(%struct.point65) align 2 %1) #0 {
  %3 = bitcast %struct.point65* %0 to i8*
  %4 = bitcast %struct.point65* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo66(%struct.point66* noalias sret(%struct.point66) align 2 %0, %struct.point66* noundef byval(%struct.point66) align 2 %1) #0 {
  %3 = bitcast %struct.point66* %0 to i8*
  %4 = bitcast %struct.point66* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo67(%struct.point67* noalias sret(%struct.point67) align 2 %0, %struct.point67* noundef byval(%struct.point67) align 2 %1) #0 {
  %3 = bitcast %struct.point67* %0 to i8*
  %4 = bitcast %struct.point67* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo68(%struct.point68* noalias sret(%struct.point68) align 4 %0, %struct.point68* noundef byval(%struct.point68) align 4 %1) #0 {
  %3 = bitcast %struct.point68* %0 to i8*
  %4 = bitcast %struct.point68* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo69(%struct.point69* noalias sret(%struct.point69) align 4 %0, %struct.point69* noundef byval(%struct.point69) align 4 %1) #0 {
  %3 = bitcast %struct.point69* %0 to i8*
  %4 = bitcast %struct.point69* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo70(%struct.point70* noalias sret(%struct.point70) align 2 %0, %struct.point70* noundef byval(%struct.point70) align 2 %1) #0 {
  %3 = bitcast %struct.point70* %0 to i8*
  %4 = bitcast %struct.point70* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo71(%struct.point71* noalias sret(%struct.point71) align 2 %0, %struct.point71* noundef byval(%struct.point71) align 2 %1) #0 {
  %3 = bitcast %struct.point71* %0 to i8*
  %4 = bitcast %struct.point71* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo72(%struct.point72* noalias sret(%struct.point72) align 2 %0, %struct.point72* noundef byval(%struct.point72) align 2 %1) #0 {
  %3 = bitcast %struct.point72* %0 to i8*
  %4 = bitcast %struct.point72* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo73(%struct.point73* noalias sret(%struct.point73) align 4 %0, %struct.point73* noundef byval(%struct.point73) align 4 %1) #0 {
  %3 = bitcast %struct.point73* %0 to i8*
  %4 = bitcast %struct.point73* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo74(%struct.point74* noalias sret(%struct.point74) align 4 %0, %struct.point74* noundef byval(%struct.point74) align 4 %1) #0 {
  %3 = bitcast %struct.point74* %0 to i8*
  %4 = bitcast %struct.point74* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo75(%struct.point75* noalias sret(%struct.point75) align 4 %0, %struct.point75* noundef byval(%struct.point75) align 4 %1) #0 {
  %3 = bitcast %struct.point75* %0 to i8*
  %4 = bitcast %struct.point75* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo76(%struct.point76* noalias sret(%struct.point76) align 4 %0, %struct.point76* noundef byval(%struct.point76) align 4 %1) #0 {
  %3 = bitcast %struct.point76* %0 to i8*
  %4 = bitcast %struct.point76* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo77(%struct.point77* noalias sret(%struct.point77) align 4 %0, %struct.point77* noundef byval(%struct.point77) align 4 %1) #0 {
  %3 = bitcast %struct.point77* %0 to i8*
  %4 = bitcast %struct.point77* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo78(%struct.point78* noalias sret(%struct.point78) align 4 %0, %struct.point78* noundef byval(%struct.point78) align 4 %1) #0 {
  %3 = bitcast %struct.point78* %0 to i8*
  %4 = bitcast %struct.point78* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo79(%struct.point79* noalias sret(%struct.point79) align 4 %0, %struct.point79* noundef byval(%struct.point79) align 4 %1) #0 {
  %3 = bitcast %struct.point79* %0 to i8*
  %4 = bitcast %struct.point79* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo80(%struct.point80* noalias sret(%struct.point80) align 4 %0, %struct.point80* noundef byval(%struct.point80) align 4 %1) #0 {
  %3 = bitcast %struct.point80* %0 to i8*
  %4 = bitcast %struct.point80* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo81(%struct.point81* noalias sret(%struct.point81) align 4 %0, %struct.point81* noundef byval(%struct.point81) align 4 %1) #0 {
  %3 = bitcast %struct.point81* %0 to i8*
  %4 = bitcast %struct.point81* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo82(%struct.point82* noalias sret(%struct.point82) align 4 %0, %struct.point82* noundef byval(%struct.point82) align 4 %1) #0 {
  %3 = bitcast %struct.point82* %0 to i8*
  %4 = bitcast %struct.point82* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo83(%struct.point83* noalias sret(%struct.point83) align 4 %0, %struct.point83* noundef byval(%struct.point83) align 4 %1) #0 {
  %3 = bitcast %struct.point83* %0 to i8*
  %4 = bitcast %struct.point83* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo84(%struct.point84* noalias sret(%struct.point84) align 4 %0, %struct.point84* noundef byval(%struct.point84) align 4 %1) #0 {
  %3 = bitcast %struct.point84* %0 to i8*
  %4 = bitcast %struct.point84* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo85(%struct.point85* noalias sret(%struct.point85) align 4 %0, %struct.point85* noundef byval(%struct.point85) align 4 %1) #0 {
  %3 = bitcast %struct.point85* %0 to i8*
  %4 = bitcast %struct.point85* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo86(%struct.point86* noalias sret(%struct.point86) align 4 %0, %struct.point86* noundef byval(%struct.point86) align 4 %1) #0 {
  %3 = bitcast %struct.point86* %0 to i8*
  %4 = bitcast %struct.point86* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo87(%struct.point87* noalias sret(%struct.point87) align 4 %0, %struct.point87* noundef byval(%struct.point87) align 4 %1) #0 {
  %3 = bitcast %struct.point87* %0 to i8*
  %4 = bitcast %struct.point87* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo88(%struct.point88* noalias sret(%struct.point88) align 4 %0, %struct.point88* noundef byval(%struct.point88) align 4 %1) #0 {
  %3 = bitcast %struct.point88* %0 to i8*
  %4 = bitcast %struct.point88* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo89(%struct.point89* noalias sret(%struct.point89) align 4 %0, %struct.point89* noundef byval(%struct.point89) align 4 %1) #0 {
  %3 = bitcast %struct.point89* %0 to i8*
  %4 = bitcast %struct.point89* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo90(%struct.point90* noalias sret(%struct.point90) align 4 %0, %struct.point90* noundef byval(%struct.point90) align 4 %1) #0 {
  %3 = bitcast %struct.point90* %0 to i8*
  %4 = bitcast %struct.point90* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo91(%struct.point91* noalias sret(%struct.point91) align 4 %0, %struct.point91* noundef byval(%struct.point91) align 4 %1) #0 {
  %3 = bitcast %struct.point91* %0 to i8*
  %4 = bitcast %struct.point91* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo92(%struct.point92* noalias sret(%struct.point92) align 4 %0, %struct.point92* noundef byval(%struct.point92) align 4 %1) #0 {
  %3 = bitcast %struct.point92* %0 to i8*
  %4 = bitcast %struct.point92* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo93(%struct.point93* noalias sret(%struct.point93) align 4 %0, %struct.point93* noundef byval(%struct.point93) align 4 %1) #0 {
  %3 = bitcast %struct.point93* %0 to i8*
  %4 = bitcast %struct.point93* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo94(%struct.point94* noalias sret(%struct.point94) align 4 %0, %struct.point94* noundef byval(%struct.point94) align 4 %1) #0 {
  %3 = bitcast %struct.point94* %0 to i8*
  %4 = bitcast %struct.point94* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo95(%struct.point95* noalias sret(%struct.point95) align 4 %0, %struct.point95* noundef byval(%struct.point95) align 4 %1) #0 {
  %3 = bitcast %struct.point95* %0 to i8*
  %4 = bitcast %struct.point95* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo96(%struct.point96* noalias sret(%struct.point96) align 4 %0, %struct.point96* noundef byval(%struct.point96) align 4 %1) #0 {
  %3 = bitcast %struct.point96* %0 to i8*
  %4 = bitcast %struct.point96* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo97(%struct.point97* noalias sret(%struct.point97) align 4 %0, %struct.point97* noundef byval(%struct.point97) align 4 %1) #0 {
  %3 = bitcast %struct.point97* %0 to i8*
  %4 = bitcast %struct.point97* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo98(%struct.point98* noalias sret(%struct.point98) align 4 %0, %struct.point98* noundef byval(%struct.point98) align 4 %1) #0 {
  %3 = bitcast %struct.point98* %0 to i8*
  %4 = bitcast %struct.point98* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo99(%struct.point99* noalias sret(%struct.point99) align 4 %0, %struct.point99* noundef byval(%struct.point99) align 4 %1) #0 {
  %3 = bitcast %struct.point99* %0 to i8*
  %4 = bitcast %struct.point99* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo100(%struct.point100* noalias sret(%struct.point100) align 4 %0, %struct.point100* noundef byval(%struct.point100) align 4 %1) #0 {
  %3 = bitcast %struct.point100* %0 to i8*
  %4 = bitcast %struct.point100* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo101(%struct.point101* noalias sret(%struct.point101) align 4 %0, %struct.point101* noundef byval(%struct.point101) align 4 %1) #0 {
  %3 = bitcast %struct.point101* %0 to i8*
  %4 = bitcast %struct.point101* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo102(%struct.point102* noalias sret(%struct.point102) align 4 %0, %struct.point102* noundef byval(%struct.point102) align 4 %1) #0 {
  %3 = bitcast %struct.point102* %0 to i8*
  %4 = bitcast %struct.point102* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo103(%struct.point103* noalias sret(%struct.point103) align 4 %0, %struct.point103* noundef byval(%struct.point103) align 4 %1) #0 {
  %3 = bitcast %struct.point103* %0 to i8*
  %4 = bitcast %struct.point103* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo104(%struct.point104* noalias sret(%struct.point104) align 4 %0, %struct.point104* noundef byval(%struct.point104) align 4 %1) #0 {
  %3 = bitcast %struct.point104* %0 to i8*
  %4 = bitcast %struct.point104* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo105(%struct.point105* noalias sret(%struct.point105) align 4 %0, %struct.point105* noundef byval(%struct.point105) align 4 %1) #0 {
  %3 = bitcast %struct.point105* %0 to i8*
  %4 = bitcast %struct.point105* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo106(%struct.point106* noalias sret(%struct.point106) align 4 %0, %struct.point106* noundef byval(%struct.point106) align 4 %1) #0 {
  %3 = bitcast %struct.point106* %0 to i8*
  %4 = bitcast %struct.point106* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo107(%struct.point107* noalias sret(%struct.point107) align 4 %0, %struct.point107* noundef byval(%struct.point107) align 4 %1) #0 {
  %3 = bitcast %struct.point107* %0 to i8*
  %4 = bitcast %struct.point107* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo108(%struct.point108* noalias sret(%struct.point108) align 4 %0, %struct.point108* noundef byval(%struct.point108) align 4 %1) #0 {
  %3 = bitcast %struct.point108* %0 to i8*
  %4 = bitcast %struct.point108* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo109(%struct.point109* noalias sret(%struct.point109) align 4 %0, %struct.point109* noundef byval(%struct.point109) align 4 %1) #0 {
  %3 = bitcast %struct.point109* %0 to i8*
  %4 = bitcast %struct.point109* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo110(%struct.point110* noalias sret(%struct.point110) align 4 %0, %struct.point110* noundef byval(%struct.point110) align 4 %1) #0 {
  %3 = bitcast %struct.point110* %0 to i8*
  %4 = bitcast %struct.point110* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo111(%struct.point111* noalias sret(%struct.point111) align 4 %0, %struct.point111* noundef byval(%struct.point111) align 4 %1) #0 {
  %3 = bitcast %struct.point111* %0 to i8*
  %4 = bitcast %struct.point111* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo112(%struct.point112* noalias sret(%struct.point112) align 4 %0, %struct.point112* noundef byval(%struct.point112) align 4 %1) #0 {
  %3 = bitcast %struct.point112* %0 to i8*
  %4 = bitcast %struct.point112* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo113(%struct.point113* noalias sret(%struct.point113) align 4 %0, %struct.point113* noundef byval(%struct.point113) align 4 %1) #0 {
  %3 = bitcast %struct.point113* %0 to i8*
  %4 = bitcast %struct.point113* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo114(%struct.point114* noalias sret(%struct.point114) align 4 %0, %struct.point114* noundef byval(%struct.point114) align 4 %1) #0 {
  %3 = bitcast %struct.point114* %0 to i8*
  %4 = bitcast %struct.point114* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo115(%struct.point115* noalias sret(%struct.point115) align 4 %0, %struct.point115* noundef byval(%struct.point115) align 4 %1) #0 {
  %3 = bitcast %struct.point115* %0 to i8*
  %4 = bitcast %struct.point115* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo116(%struct.point116* noalias sret(%struct.point116) align 4 %0, %struct.point116* noundef byval(%struct.point116) align 4 %1) #0 {
  %3 = bitcast %struct.point116* %0 to i8*
  %4 = bitcast %struct.point116* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo117(%struct.point117* noalias sret(%struct.point117) align 4 %0, %struct.point117* noundef byval(%struct.point117) align 4 %1) #0 {
  %3 = bitcast %struct.point117* %0 to i8*
  %4 = bitcast %struct.point117* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo118(%struct.point118* noalias sret(%struct.point118) align 4 %0, %struct.point118* noundef byval(%struct.point118) align 4 %1) #0 {
  %3 = bitcast %struct.point118* %0 to i8*
  %4 = bitcast %struct.point118* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo119(%struct.point119* noalias sret(%struct.point119) align 4 %0, %struct.point119* noundef byval(%struct.point119) align 4 %1) #0 {
  %3 = bitcast %struct.point119* %0 to i8*
  %4 = bitcast %struct.point119* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo120(%struct.point120* noalias sret(%struct.point120) align 4 %0, %struct.point120* noundef byval(%struct.point120) align 4 %1) #0 {
  %3 = bitcast %struct.point120* %0 to i8*
  %4 = bitcast %struct.point120* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo121(%struct.point121* noalias sret(%struct.point121) align 4 %0, %struct.point121* noundef byval(%struct.point121) align 4 %1) #0 {
  %3 = bitcast %struct.point121* %0 to i8*
  %4 = bitcast %struct.point121* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo122(%struct.point122* noalias sret(%struct.point122) align 4 %0, %struct.point122* noundef byval(%struct.point122) align 4 %1) #0 {
  %3 = bitcast %struct.point122* %0 to i8*
  %4 = bitcast %struct.point122* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo123(%struct.point123* noalias sret(%struct.point123) align 4 %0, %struct.point123* noundef byval(%struct.point123) align 4 %1) #0 {
  %3 = bitcast %struct.point123* %0 to i8*
  %4 = bitcast %struct.point123* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo124(%struct.point124* noalias sret(%struct.point124) align 4 %0, %struct.point124* noundef byval(%struct.point124) align 4 %1) #0 {
  %3 = bitcast %struct.point124* %0 to i8*
  %4 = bitcast %struct.point124* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo125(%struct.point125* noalias sret(%struct.point125) align 4 %0, %struct.point125* noundef byval(%struct.point125) align 4 %1) #0 {
  %3 = bitcast %struct.point125* %0 to i8*
  %4 = bitcast %struct.point125* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo126(%struct.point126* noalias sret(%struct.point126) align 4 %0, %struct.point126* noundef byval(%struct.point126) align 4 %1) #0 {
  %3 = bitcast %struct.point126* %0 to i8*
  %4 = bitcast %struct.point126* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo127(%struct.point127* noalias sret(%struct.point127) align 4 %0, %struct.point127* noundef byval(%struct.point127) align 4 %1) #0 {
  %3 = bitcast %struct.point127* %0 to i8*
  %4 = bitcast %struct.point127* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo128(%struct.point128* noalias sret(%struct.point128) align 4 %0, %struct.point128* noundef byval(%struct.point128) align 4 %1) #0 {
  %3 = bitcast %struct.point128* %0 to i8*
  %4 = bitcast %struct.point128* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo129(%struct.point129* noalias sret(%struct.point129) align 4 %0, %struct.point129* noundef byval(%struct.point129) align 4 %1) #0 {
  %3 = bitcast %struct.point129* %0 to i8*
  %4 = bitcast %struct.point129* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo130(%struct.point130* noalias sret(%struct.point130) align 4 %0, %struct.point130* noundef byval(%struct.point130) align 4 %1) #0 {
  %3 = bitcast %struct.point130* %0 to i8*
  %4 = bitcast %struct.point130* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo131(%struct.point131* noalias sret(%struct.point131) align 4 %0, %struct.point131* noundef byval(%struct.point131) align 4 %1) #0 {
  %3 = bitcast %struct.point131* %0 to i8*
  %4 = bitcast %struct.point131* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo132(%struct.point132* noalias sret(%struct.point132) align 4 %0, %struct.point132* noundef byval(%struct.point132) align 4 %1) #0 {
  %3 = bitcast %struct.point132* %0 to i8*
  %4 = bitcast %struct.point132* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo133(%struct.point133* noalias sret(%struct.point133) align 4 %0, %struct.point133* noundef byval(%struct.point133) align 4 %1) #0 {
  %3 = bitcast %struct.point133* %0 to i8*
  %4 = bitcast %struct.point133* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo134(%struct.point134* noalias sret(%struct.point134) align 4 %0, %struct.point134* noundef byval(%struct.point134) align 4 %1) #0 {
  %3 = bitcast %struct.point134* %0 to i8*
  %4 = bitcast %struct.point134* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo135(%struct.point135* noalias sret(%struct.point135) align 4 %0, %struct.point135* noundef byval(%struct.point135) align 4 %1) #0 {
  %3 = bitcast %struct.point135* %0 to i8*
  %4 = bitcast %struct.point135* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo136(%struct.point136* noalias sret(%struct.point136) align 4 %0, %struct.point136* noundef byval(%struct.point136) align 4 %1) #0 {
  %3 = bitcast %struct.point136* %0 to i8*
  %4 = bitcast %struct.point136* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo137(%struct.point137* noalias sret(%struct.point137) align 4 %0, %struct.point137* noundef byval(%struct.point137) align 4 %1) #0 {
  %3 = bitcast %struct.point137* %0 to i8*
  %4 = bitcast %struct.point137* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo138(%struct.point138* noalias sret(%struct.point138) align 4 %0, %struct.point138* noundef byval(%struct.point138) align 4 %1) #0 {
  %3 = bitcast %struct.point138* %0 to i8*
  %4 = bitcast %struct.point138* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo139(%struct.point139* noalias sret(%struct.point139) align 4 %0, %struct.point139* noundef byval(%struct.point139) align 4 %1) #0 {
  %3 = bitcast %struct.point139* %0 to i8*
  %4 = bitcast %struct.point139* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo140(%struct.point140* noalias sret(%struct.point140) align 4 %0, %struct.point140* noundef byval(%struct.point140) align 4 %1) #0 {
  %3 = bitcast %struct.point140* %0 to i8*
  %4 = bitcast %struct.point140* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo141(%struct.point141* noalias sret(%struct.point141) align 4 %0, %struct.point141* noundef byval(%struct.point141) align 4 %1) #0 {
  %3 = bitcast %struct.point141* %0 to i8*
  %4 = bitcast %struct.point141* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo142(%struct.point142* noalias sret(%struct.point142) align 4 %0, %struct.point142* noundef byval(%struct.point142) align 4 %1) #0 {
  %3 = bitcast %struct.point142* %0 to i8*
  %4 = bitcast %struct.point142* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo143(%struct.point143* noalias sret(%struct.point143) align 4 %0, %struct.point143* noundef byval(%struct.point143) align 4 %1) #0 {
  %3 = bitcast %struct.point143* %0 to i8*
  %4 = bitcast %struct.point143* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo144(%struct.point144* noalias sret(%struct.point144) align 4 %0, %struct.point144* noundef byval(%struct.point144) align 4 %1) #0 {
  %3 = bitcast %struct.point144* %0 to i8*
  %4 = bitcast %struct.point144* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo145(%struct.point145* noalias sret(%struct.point145) align 4 %0, %struct.point145* noundef byval(%struct.point145) align 4 %1) #0 {
  %3 = bitcast %struct.point145* %0 to i8*
  %4 = bitcast %struct.point145* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo146(%struct.point146* noalias sret(%struct.point146) align 4 %0, %struct.point146* noundef byval(%struct.point146) align 4 %1) #0 {
  %3 = bitcast %struct.point146* %0 to i8*
  %4 = bitcast %struct.point146* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo147(%struct.point147* noalias sret(%struct.point147) align 4 %0, %struct.point147* noundef byval(%struct.point147) align 4 %1) #0 {
  %3 = bitcast %struct.point147* %0 to i8*
  %4 = bitcast %struct.point147* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo148(%struct.point148* noalias sret(%struct.point148) align 4 %0, %struct.point148* noundef byval(%struct.point148) align 4 %1) #0 {
  %3 = bitcast %struct.point148* %0 to i8*
  %4 = bitcast %struct.point148* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo149(%struct.point149* noalias sret(%struct.point149) align 4 %0, %struct.point149* noundef byval(%struct.point149) align 4 %1) #0 {
  %3 = bitcast %struct.point149* %0 to i8*
  %4 = bitcast %struct.point149* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo150(%struct.point150* noalias sret(%struct.point150) align 4 %0, %struct.point150* noundef byval(%struct.point150) align 4 %1) #0 {
  %3 = bitcast %struct.point150* %0 to i8*
  %4 = bitcast %struct.point150* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo151(%struct.point151* noalias sret(%struct.point151) align 4 %0, %struct.point151* noundef byval(%struct.point151) align 4 %1) #0 {
  %3 = bitcast %struct.point151* %0 to i8*
  %4 = bitcast %struct.point151* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo152(%struct.point152* noalias sret(%struct.point152) align 4 %0, %struct.point152* noundef byval(%struct.point152) align 4 %1) #0 {
  %3 = bitcast %struct.point152* %0 to i8*
  %4 = bitcast %struct.point152* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo153(%struct.point153* noalias sret(%struct.point153) align 4 %0, %struct.point153* noundef byval(%struct.point153) align 4 %1) #0 {
  %3 = bitcast %struct.point153* %0 to i8*
  %4 = bitcast %struct.point153* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo154(%struct.point154* noalias sret(%struct.point154) align 4 %0, %struct.point154* noundef byval(%struct.point154) align 4 %1) #0 {
  %3 = bitcast %struct.point154* %0 to i8*
  %4 = bitcast %struct.point154* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo155(%struct.point155* noalias sret(%struct.point155) align 4 %0, %struct.point155* noundef byval(%struct.point155) align 4 %1) #0 {
  %3 = bitcast %struct.point155* %0 to i8*
  %4 = bitcast %struct.point155* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo156(%struct.point156* noalias sret(%struct.point156) align 4 %0, %struct.point156* noundef byval(%struct.point156) align 4 %1) #0 {
  %3 = bitcast %struct.point156* %0 to i8*
  %4 = bitcast %struct.point156* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo157(%struct.point157* noalias sret(%struct.point157) align 4 %0, %struct.point157* noundef byval(%struct.point157) align 4 %1) #0 {
  %3 = bitcast %struct.point157* %0 to i8*
  %4 = bitcast %struct.point157* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo158(%struct.point158* noalias sret(%struct.point158) align 4 %0, %struct.point158* noundef byval(%struct.point158) align 4 %1) #0 {
  %3 = bitcast %struct.point158* %0 to i8*
  %4 = bitcast %struct.point158* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo159(%struct.point159* noalias sret(%struct.point159) align 4 %0, %struct.point159* noundef byval(%struct.point159) align 4 %1) #0 {
  %3 = bitcast %struct.point159* %0 to i8*
  %4 = bitcast %struct.point159* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo160(%struct.point160* noalias sret(%struct.point160) align 4 %0, %struct.point160* noundef byval(%struct.point160) align 4 %1) #0 {
  %3 = bitcast %struct.point160* %0 to i8*
  %4 = bitcast %struct.point160* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo161(%struct.point161* noalias sret(%struct.point161) align 4 %0, %struct.point161* noundef byval(%struct.point161) align 4 %1) #0 {
  %3 = bitcast %struct.point161* %0 to i8*
  %4 = bitcast %struct.point161* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo162(%struct.point162* noalias sret(%struct.point162) align 4 %0, %struct.point162* noundef byval(%struct.point162) align 4 %1) #0 {
  %3 = bitcast %struct.point162* %0 to i8*
  %4 = bitcast %struct.point162* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo163(%struct.point163* noalias sret(%struct.point163) align 4 %0, %struct.point163* noundef byval(%struct.point163) align 4 %1) #0 {
  %3 = bitcast %struct.point163* %0 to i8*
  %4 = bitcast %struct.point163* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo164(%struct.point164* noalias sret(%struct.point164) align 4 %0, %struct.point164* noundef byval(%struct.point164) align 4 %1) #0 {
  %3 = bitcast %struct.point164* %0 to i8*
  %4 = bitcast %struct.point164* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo165(%struct.point165* noalias sret(%struct.point165) align 4 %0, %struct.point165* noundef byval(%struct.point165) align 4 %1) #0 {
  %3 = bitcast %struct.point165* %0 to i8*
  %4 = bitcast %struct.point165* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo166(%struct.point166* noalias sret(%struct.point166) align 4 %0, %struct.point166* noundef byval(%struct.point166) align 4 %1) #0 {
  %3 = bitcast %struct.point166* %0 to i8*
  %4 = bitcast %struct.point166* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo167(%struct.point167* noalias sret(%struct.point167) align 4 %0, %struct.point167* noundef byval(%struct.point167) align 4 %1) #0 {
  %3 = bitcast %struct.point167* %0 to i8*
  %4 = bitcast %struct.point167* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo168(%struct.point168* noalias sret(%struct.point168) align 4 %0, %struct.point168* noundef byval(%struct.point168) align 4 %1) #0 {
  %3 = bitcast %struct.point168* %0 to i8*
  %4 = bitcast %struct.point168* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo169(%struct.point169* noalias sret(%struct.point169) align 4 %0, %struct.point169* noundef byval(%struct.point169) align 4 %1) #0 {
  %3 = bitcast %struct.point169* %0 to i8*
  %4 = bitcast %struct.point169* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo170(%struct.point170* noalias sret(%struct.point170) align 4 %0, %struct.point170* noundef byval(%struct.point170) align 4 %1) #0 {
  %3 = bitcast %struct.point170* %0 to i8*
  %4 = bitcast %struct.point170* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo171(%struct.point171* noalias sret(%struct.point171) align 4 %0, %struct.point171* noundef byval(%struct.point171) align 4 %1) #0 {
  %3 = bitcast %struct.point171* %0 to i8*
  %4 = bitcast %struct.point171* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo172(%struct.point172* noalias sret(%struct.point172) align 4 %0, %struct.point172* noundef byval(%struct.point172) align 4 %1) #0 {
  %3 = bitcast %struct.point172* %0 to i8*
  %4 = bitcast %struct.point172* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo173(%struct.point173* noalias sret(%struct.point173) align 4 %0, %struct.point173* noundef byval(%struct.point173) align 4 %1) #0 {
  %3 = bitcast %struct.point173* %0 to i8*
  %4 = bitcast %struct.point173* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo174(%struct.point174* noalias sret(%struct.point174) align 4 %0, %struct.point174* noundef byval(%struct.point174) align 4 %1) #0 {
  %3 = bitcast %struct.point174* %0 to i8*
  %4 = bitcast %struct.point174* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo175(%struct.point175* noalias sret(%struct.point175) align 4 %0, %struct.point175* noundef byval(%struct.point175) align 4 %1) #0 {
  %3 = bitcast %struct.point175* %0 to i8*
  %4 = bitcast %struct.point175* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo176(%struct.point176* noalias sret(%struct.point176) align 4 %0, %struct.point176* noundef byval(%struct.point176) align 4 %1) #0 {
  %3 = bitcast %struct.point176* %0 to i8*
  %4 = bitcast %struct.point176* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo177(%struct.point177* noalias sret(%struct.point177) align 4 %0, %struct.point177* noundef byval(%struct.point177) align 4 %1) #0 {
  %3 = bitcast %struct.point177* %0 to i8*
  %4 = bitcast %struct.point177* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo178(%struct.point178* noalias sret(%struct.point178) align 4 %0, %struct.point178* noundef byval(%struct.point178) align 4 %1) #0 {
  %3 = bitcast %struct.point178* %0 to i8*
  %4 = bitcast %struct.point178* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo179(%struct.point179* noalias sret(%struct.point179) align 4 %0, %struct.point179* noundef byval(%struct.point179) align 4 %1) #0 {
  %3 = bitcast %struct.point179* %0 to i8*
  %4 = bitcast %struct.point179* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo180(%struct.point180* noalias sret(%struct.point180) align 4 %0, %struct.point180* noundef byval(%struct.point180) align 4 %1) #0 {
  %3 = bitcast %struct.point180* %0 to i8*
  %4 = bitcast %struct.point180* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo181(%struct.point181* noalias sret(%struct.point181) align 4 %0, %struct.point181* noundef byval(%struct.point181) align 4 %1) #0 {
  %3 = bitcast %struct.point181* %0 to i8*
  %4 = bitcast %struct.point181* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo182(%struct.point182* noalias sret(%struct.point182) align 4 %0, %struct.point182* noundef byval(%struct.point182) align 4 %1) #0 {
  %3 = bitcast %struct.point182* %0 to i8*
  %4 = bitcast %struct.point182* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo183(%struct.point183* noalias sret(%struct.point183) align 4 %0, %struct.point183* noundef byval(%struct.point183) align 4 %1) #0 {
  %3 = bitcast %struct.point183* %0 to i8*
  %4 = bitcast %struct.point183* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo184(%struct.point184* noalias sret(%struct.point184) align 4 %0, %struct.point184* noundef byval(%struct.point184) align 4 %1) #0 {
  %3 = bitcast %struct.point184* %0 to i8*
  %4 = bitcast %struct.point184* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo185(%struct.point185* noalias sret(%struct.point185) align 4 %0, %struct.point185* noundef byval(%struct.point185) align 4 %1) #0 {
  %3 = bitcast %struct.point185* %0 to i8*
  %4 = bitcast %struct.point185* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo186(%struct.point186* noalias sret(%struct.point186) align 4 %0, %struct.point186* noundef byval(%struct.point186) align 4 %1) #0 {
  %3 = bitcast %struct.point186* %0 to i8*
  %4 = bitcast %struct.point186* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo187(%struct.point187* noalias sret(%struct.point187) align 4 %0, %struct.point187* noundef byval(%struct.point187) align 4 %1) #0 {
  %3 = bitcast %struct.point187* %0 to i8*
  %4 = bitcast %struct.point187* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo188(%struct.point188* noalias sret(%struct.point188) align 4 %0, %struct.point188* noundef byval(%struct.point188) align 4 %1) #0 {
  %3 = bitcast %struct.point188* %0 to i8*
  %4 = bitcast %struct.point188* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo189(%struct.point189* noalias sret(%struct.point189) align 4 %0, %struct.point189* noundef byval(%struct.point189) align 4 %1) #0 {
  %3 = bitcast %struct.point189* %0 to i8*
  %4 = bitcast %struct.point189* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo190(%struct.point190* noalias sret(%struct.point190) align 4 %0, %struct.point190* noundef byval(%struct.point190) align 4 %1) #0 {
  %3 = bitcast %struct.point190* %0 to i8*
  %4 = bitcast %struct.point190* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo191(%struct.point191* noalias sret(%struct.point191) align 4 %0, %struct.point191* noundef byval(%struct.point191) align 4 %1) #0 {
  %3 = bitcast %struct.point191* %0 to i8*
  %4 = bitcast %struct.point191* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo192(%struct.point192* noalias sret(%struct.point192) align 4 %0, %struct.point192* noundef byval(%struct.point192) align 4 %1) #0 {
  %3 = bitcast %struct.point192* %0 to i8*
  %4 = bitcast %struct.point192* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo193(%struct.point193* noalias sret(%struct.point193) align 4 %0, %struct.point193* noundef byval(%struct.point193) align 4 %1) #0 {
  %3 = bitcast %struct.point193* %0 to i8*
  %4 = bitcast %struct.point193* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo194(%struct.point194* noalias sret(%struct.point194) align 4 %0, %struct.point194* noundef byval(%struct.point194) align 4 %1) #0 {
  %3 = bitcast %struct.point194* %0 to i8*
  %4 = bitcast %struct.point194* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo195(%struct.point195* noalias sret(%struct.point195) align 4 %0, %struct.point195* noundef byval(%struct.point195) align 4 %1) #0 {
  %3 = bitcast %struct.point195* %0 to i8*
  %4 = bitcast %struct.point195* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo196(%struct.point196* noalias sret(%struct.point196) align 4 %0, %struct.point196* noundef byval(%struct.point196) align 4 %1) #0 {
  %3 = bitcast %struct.point196* %0 to i8*
  %4 = bitcast %struct.point196* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo197(%struct.point197* noalias sret(%struct.point197) align 4 %0, %struct.point197* noundef byval(%struct.point197) align 4 %1) #0 {
  %3 = bitcast %struct.point197* %0 to i8*
  %4 = bitcast %struct.point197* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo198(%struct.point198* noalias sret(%struct.point198) align 4 %0, %struct.point198* noundef byval(%struct.point198) align 4 %1) #0 {
  %3 = bitcast %struct.point198* %0 to i8*
  %4 = bitcast %struct.point198* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo199(%struct.point199* noalias sret(%struct.point199) align 4 %0, %struct.point199* noundef byval(%struct.point199) align 4 %1) #0 {
  %3 = bitcast %struct.point199* %0 to i8*
  %4 = bitcast %struct.point199* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo200(%struct.point200* noalias sret(%struct.point200) align 4 %0, %struct.point200* noundef byval(%struct.point200) align 4 %1) #0 {
  %3 = bitcast %struct.point200* %0 to i8*
  %4 = bitcast %struct.point200* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo201(%struct.point201* noalias sret(%struct.point201) align 4 %0, %struct.point201* noundef byval(%struct.point201) align 4 %1) #0 {
  %3 = bitcast %struct.point201* %0 to i8*
  %4 = bitcast %struct.point201* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo202(%struct.point202* noalias sret(%struct.point202) align 4 %0, %struct.point202* noundef byval(%struct.point202) align 4 %1) #0 {
  %3 = bitcast %struct.point202* %0 to i8*
  %4 = bitcast %struct.point202* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo203(%struct.point203* noalias sret(%struct.point203) align 4 %0, %struct.point203* noundef byval(%struct.point203) align 4 %1) #0 {
  %3 = bitcast %struct.point203* %0 to i8*
  %4 = bitcast %struct.point203* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo204(%struct.point204* noalias sret(%struct.point204) align 4 %0, %struct.point204* noundef byval(%struct.point204) align 4 %1) #0 {
  %3 = bitcast %struct.point204* %0 to i8*
  %4 = bitcast %struct.point204* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo205(%struct.point205* noalias sret(%struct.point205) align 4 %0, %struct.point205* noundef byval(%struct.point205) align 4 %1) #0 {
  %3 = bitcast %struct.point205* %0 to i8*
  %4 = bitcast %struct.point205* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo206(%struct.point206* noalias sret(%struct.point206) align 4 %0, %struct.point206* noundef byval(%struct.point206) align 4 %1) #0 {
  %3 = bitcast %struct.point206* %0 to i8*
  %4 = bitcast %struct.point206* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo207(%struct.point207* noalias sret(%struct.point207) align 4 %0, %struct.point207* noundef byval(%struct.point207) align 4 %1) #0 {
  %3 = bitcast %struct.point207* %0 to i8*
  %4 = bitcast %struct.point207* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo208(%struct.point208* noalias sret(%struct.point208) align 4 %0, %struct.point208* noundef byval(%struct.point208) align 4 %1) #0 {
  %3 = bitcast %struct.point208* %0 to i8*
  %4 = bitcast %struct.point208* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo209(%struct.point209* noalias sret(%struct.point209) align 4 %0, %struct.point209* noundef byval(%struct.point209) align 4 %1) #0 {
  %3 = bitcast %struct.point209* %0 to i8*
  %4 = bitcast %struct.point209* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo210(%struct.point210* noalias sret(%struct.point210) align 4 %0, %struct.point210* noundef byval(%struct.point210) align 4 %1) #0 {
  %3 = bitcast %struct.point210* %0 to i8*
  %4 = bitcast %struct.point210* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo211(%struct.point211* noalias sret(%struct.point211) align 4 %0, %struct.point211* noundef byval(%struct.point211) align 4 %1) #0 {
  %3 = bitcast %struct.point211* %0 to i8*
  %4 = bitcast %struct.point211* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo212(%struct.point212* noalias sret(%struct.point212) align 4 %0, %struct.point212* noundef byval(%struct.point212) align 4 %1) #0 {
  %3 = bitcast %struct.point212* %0 to i8*
  %4 = bitcast %struct.point212* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo213(%struct.point213* noalias sret(%struct.point213) align 4 %0, %struct.point213* noundef byval(%struct.point213) align 4 %1) #0 {
  %3 = bitcast %struct.point213* %0 to i8*
  %4 = bitcast %struct.point213* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo214(%struct.point214* noalias sret(%struct.point214) align 4 %0, %struct.point214* noundef byval(%struct.point214) align 4 %1) #0 {
  %3 = bitcast %struct.point214* %0 to i8*
  %4 = bitcast %struct.point214* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo215(%struct.point215* noalias sret(%struct.point215) align 4 %0, %struct.point215* noundef byval(%struct.point215) align 4 %1) #0 {
  %3 = bitcast %struct.point215* %0 to i8*
  %4 = bitcast %struct.point215* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo216(%struct.point216* noalias sret(%struct.point216) align 4 %0, %struct.point216* noundef byval(%struct.point216) align 4 %1) #0 {
  %3 = bitcast %struct.point216* %0 to i8*
  %4 = bitcast %struct.point216* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo217(%struct.point217* noalias sret(%struct.point217) align 4 %0, %struct.point217* noundef byval(%struct.point217) align 4 %1) #0 {
  %3 = bitcast %struct.point217* %0 to i8*
  %4 = bitcast %struct.point217* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo218(%struct.point218* noalias sret(%struct.point218) align 4 %0, %struct.point218* noundef byval(%struct.point218) align 4 %1) #0 {
  %3 = bitcast %struct.point218* %0 to i8*
  %4 = bitcast %struct.point218* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo219(%struct.point219* noalias sret(%struct.point219) align 4 %0, %struct.point219* noundef byval(%struct.point219) align 4 %1) #0 {
  %3 = bitcast %struct.point219* %0 to i8*
  %4 = bitcast %struct.point219* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo220(%struct.point220* noalias sret(%struct.point220) align 4 %0, %struct.point220* noundef byval(%struct.point220) align 4 %1) #0 {
  %3 = bitcast %struct.point220* %0 to i8*
  %4 = bitcast %struct.point220* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo221(%struct.point221* noalias sret(%struct.point221) align 4 %0, %struct.point221* noundef byval(%struct.point221) align 4 %1) #0 {
  %3 = bitcast %struct.point221* %0 to i8*
  %4 = bitcast %struct.point221* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo222(%struct.point222* noalias sret(%struct.point222) align 4 %0, %struct.point222* noundef byval(%struct.point222) align 4 %1) #0 {
  %3 = bitcast %struct.point222* %0 to i8*
  %4 = bitcast %struct.point222* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo223(%struct.point223* noalias sret(%struct.point223) align 4 %0, %struct.point223* noundef byval(%struct.point223) align 4 %1) #0 {
  %3 = bitcast %struct.point223* %0 to i8*
  %4 = bitcast %struct.point223* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo224(%struct.point224* noalias sret(%struct.point224) align 4 %0, %struct.point224* noundef byval(%struct.point224) align 4 %1) #0 {
  %3 = bitcast %struct.point224* %0 to i8*
  %4 = bitcast %struct.point224* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo225(%struct.point225* noalias sret(%struct.point225) align 4 %0, %struct.point225* noundef byval(%struct.point225) align 4 %1) #0 {
  %3 = bitcast %struct.point225* %0 to i8*
  %4 = bitcast %struct.point225* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo226(%struct.point226* noalias sret(%struct.point226) align 4 %0, %struct.point226* noundef byval(%struct.point226) align 4 %1) #0 {
  %3 = bitcast %struct.point226* %0 to i8*
  %4 = bitcast %struct.point226* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo227(%struct.point227* noalias sret(%struct.point227) align 4 %0, %struct.point227* noundef byval(%struct.point227) align 4 %1) #0 {
  %3 = bitcast %struct.point227* %0 to i8*
  %4 = bitcast %struct.point227* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo228(%struct.point228* noalias sret(%struct.point228) align 4 %0, %struct.point228* noundef byval(%struct.point228) align 4 %1) #0 {
  %3 = bitcast %struct.point228* %0 to i8*
  %4 = bitcast %struct.point228* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo229(%struct.point229* noalias sret(%struct.point229) align 4 %0, %struct.point229* noundef byval(%struct.point229) align 4 %1) #0 {
  %3 = bitcast %struct.point229* %0 to i8*
  %4 = bitcast %struct.point229* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo230(%struct.point230* noalias sret(%struct.point230) align 4 %0, %struct.point230* noundef byval(%struct.point230) align 4 %1) #0 {
  %3 = bitcast %struct.point230* %0 to i8*
  %4 = bitcast %struct.point230* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo231(%struct.point231* noalias sret(%struct.point231) align 4 %0, %struct.point231* noundef byval(%struct.point231) align 4 %1) #0 {
  %3 = bitcast %struct.point231* %0 to i8*
  %4 = bitcast %struct.point231* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo232(%struct.point232* noalias sret(%struct.point232) align 4 %0, %struct.point232* noundef byval(%struct.point232) align 4 %1) #0 {
  %3 = bitcast %struct.point232* %0 to i8*
  %4 = bitcast %struct.point232* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo233(%struct.point233* noalias sret(%struct.point233) align 4 %0, %struct.point233* noundef byval(%struct.point233) align 4 %1) #0 {
  %3 = bitcast %struct.point233* %0 to i8*
  %4 = bitcast %struct.point233* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo234(%struct.point234* noalias sret(%struct.point234) align 4 %0, %struct.point234* noundef byval(%struct.point234) align 4 %1) #0 {
  %3 = bitcast %struct.point234* %0 to i8*
  %4 = bitcast %struct.point234* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo235(%struct.point235* noalias sret(%struct.point235) align 4 %0, %struct.point235* noundef byval(%struct.point235) align 4 %1) #0 {
  %3 = bitcast %struct.point235* %0 to i8*
  %4 = bitcast %struct.point235* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo236(%struct.point236* noalias sret(%struct.point236) align 4 %0, %struct.point236* noundef byval(%struct.point236) align 4 %1) #0 {
  %3 = bitcast %struct.point236* %0 to i8*
  %4 = bitcast %struct.point236* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo237(%struct.point237* noalias sret(%struct.point237) align 4 %0, %struct.point237* noundef byval(%struct.point237) align 4 %1) #0 {
  %3 = bitcast %struct.point237* %0 to i8*
  %4 = bitcast %struct.point237* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo238(%struct.point238* noalias sret(%struct.point238) align 4 %0, %struct.point238* noundef byval(%struct.point238) align 4 %1) #0 {
  %3 = bitcast %struct.point238* %0 to i8*
  %4 = bitcast %struct.point238* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo239(%struct.point239* noalias sret(%struct.point239) align 4 %0, %struct.point239* noundef byval(%struct.point239) align 4 %1) #0 {
  %3 = bitcast %struct.point239* %0 to i8*
  %4 = bitcast %struct.point239* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo240(%struct.point240* noalias sret(%struct.point240) align 4 %0, %struct.point240* noundef byval(%struct.point240) align 4 %1) #0 {
  %3 = bitcast %struct.point240* %0 to i8*
  %4 = bitcast %struct.point240* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo241(%struct.point241* noalias sret(%struct.point241) align 4 %0, %struct.point241* noundef byval(%struct.point241) align 4 %1) #0 {
  %3 = bitcast %struct.point241* %0 to i8*
  %4 = bitcast %struct.point241* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo242(%struct.point242* noalias sret(%struct.point242) align 4 %0, %struct.point242* noundef byval(%struct.point242) align 4 %1) #0 {
  %3 = bitcast %struct.point242* %0 to i8*
  %4 = bitcast %struct.point242* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo243(%struct.point243* noalias sret(%struct.point243) align 4 %0, %struct.point243* noundef byval(%struct.point243) align 4 %1) #0 {
  %3 = bitcast %struct.point243* %0 to i8*
  %4 = bitcast %struct.point243* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo244(%struct.point244* noalias sret(%struct.point244) align 4 %0, %struct.point244* noundef byval(%struct.point244) align 4 %1) #0 {
  %3 = bitcast %struct.point244* %0 to i8*
  %4 = bitcast %struct.point244* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo245(%struct.point245* noalias sret(%struct.point245) align 4 %0, %struct.point245* noundef byval(%struct.point245) align 4 %1) #0 {
  %3 = bitcast %struct.point245* %0 to i8*
  %4 = bitcast %struct.point245* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo246(%struct.point246* noalias sret(%struct.point246) align 4 %0, %struct.point246* noundef byval(%struct.point246) align 4 %1) #0 {
  %3 = bitcast %struct.point246* %0 to i8*
  %4 = bitcast %struct.point246* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo247(%struct.point247* noalias sret(%struct.point247) align 4 %0, %struct.point247* noundef byval(%struct.point247) align 4 %1) #0 {
  %3 = bitcast %struct.point247* %0 to i8*
  %4 = bitcast %struct.point247* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo248(%struct.point248* noalias sret(%struct.point248) align 4 %0, %struct.point248* noundef byval(%struct.point248) align 4 %1) #0 {
  %3 = bitcast %struct.point248* %0 to i8*
  %4 = bitcast %struct.point248* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo249(%struct.point249* noalias sret(%struct.point249) align 4 %0, %struct.point249* noundef byval(%struct.point249) align 4 %1) #0 {
  %3 = bitcast %struct.point249* %0 to i8*
  %4 = bitcast %struct.point249* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo250(%struct.point250* noalias sret(%struct.point250) align 4 %0, %struct.point250* noundef byval(%struct.point250) align 4 %1) #0 {
  %3 = bitcast %struct.point250* %0 to i8*
  %4 = bitcast %struct.point250* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo251(%struct.point251* noalias sret(%struct.point251) align 4 %0, %struct.point251* noundef byval(%struct.point251) align 4 %1) #0 {
  %3 = bitcast %struct.point251* %0 to i8*
  %4 = bitcast %struct.point251* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo252(%struct.point252* noalias sret(%struct.point252) align 4 %0, %struct.point252* noundef byval(%struct.point252) align 4 %1) #0 {
  %3 = bitcast %struct.point252* %0 to i8*
  %4 = bitcast %struct.point252* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo253(%struct.point253* noalias sret(%struct.point253) align 4 %0, %struct.point253* noundef byval(%struct.point253) align 4 %1) #0 {
  %3 = bitcast %struct.point253* %0 to i8*
  %4 = bitcast %struct.point253* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo254(%struct.point254* noalias sret(%struct.point254) align 4 %0, %struct.point254* noundef byval(%struct.point254) align 4 %1) #0 {
  %3 = bitcast %struct.point254* %0 to i8*
  %4 = bitcast %struct.point254* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo255(%struct.point255* noalias sret(%struct.point255) align 4 %0, %struct.point255* noundef byval(%struct.point255) align 4 %1) #0 {
  %3 = bitcast %struct.point255* %0 to i8*
  %4 = bitcast %struct.point255* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo256(%struct.point256* noalias sret(%struct.point256) align 4 %0, %struct.point256* noundef byval(%struct.point256) align 4 %1) #0 {
  %3 = bitcast %struct.point256* %0 to i8*
  %4 = bitcast %struct.point256* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo257(%struct.point257* noalias sret(%struct.point257) align 4 %0, %struct.point257* noundef byval(%struct.point257) align 4 %1) #0 {
  %3 = bitcast %struct.point257* %0 to i8*
  %4 = bitcast %struct.point257* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo258(%struct.point258* noalias sret(%struct.point258) align 4 %0, %struct.point258* noundef byval(%struct.point258) align 4 %1) #0 {
  %3 = bitcast %struct.point258* %0 to i8*
  %4 = bitcast %struct.point258* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo259(%struct.point259* noalias sret(%struct.point259) align 4 %0, %struct.point259* noundef byval(%struct.point259) align 4 %1) #0 {
  %3 = bitcast %struct.point259* %0 to i8*
  %4 = bitcast %struct.point259* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo260(%struct.point260* noalias sret(%struct.point260) align 4 %0, %struct.point260* noundef byval(%struct.point260) align 4 %1) #0 {
  %3 = bitcast %struct.point260* %0 to i8*
  %4 = bitcast %struct.point260* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo261(%struct.point261* noalias sret(%struct.point261) align 4 %0, %struct.point261* noundef byval(%struct.point261) align 4 %1) #0 {
  %3 = bitcast %struct.point261* %0 to i8*
  %4 = bitcast %struct.point261* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo262(%struct.point262* noalias sret(%struct.point262) align 4 %0, %struct.point262* noundef byval(%struct.point262) align 4 %1) #0 {
  %3 = bitcast %struct.point262* %0 to i8*
  %4 = bitcast %struct.point262* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo263(%struct.point263* noalias sret(%struct.point263) align 4 %0, %struct.point263* noundef byval(%struct.point263) align 4 %1) #0 {
  %3 = bitcast %struct.point263* %0 to i8*
  %4 = bitcast %struct.point263* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo264(%struct.point264* noalias sret(%struct.point264) align 4 %0, %struct.point264* noundef byval(%struct.point264) align 4 %1) #0 {
  %3 = bitcast %struct.point264* %0 to i8*
  %4 = bitcast %struct.point264* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo265(%struct.point265* noalias sret(%struct.point265) align 4 %0, %struct.point265* noundef byval(%struct.point265) align 4 %1) #0 {
  %3 = bitcast %struct.point265* %0 to i8*
  %4 = bitcast %struct.point265* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo266(%struct.point266* noalias sret(%struct.point266) align 4 %0, %struct.point266* noundef byval(%struct.point266) align 4 %1) #0 {
  %3 = bitcast %struct.point266* %0 to i8*
  %4 = bitcast %struct.point266* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo267(%struct.point267* noalias sret(%struct.point267) align 4 %0, %struct.point267* noundef byval(%struct.point267) align 4 %1) #0 {
  %3 = bitcast %struct.point267* %0 to i8*
  %4 = bitcast %struct.point267* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo268(%struct.point268* noalias sret(%struct.point268) align 4 %0, %struct.point268* noundef byval(%struct.point268) align 4 %1) #0 {
  %3 = bitcast %struct.point268* %0 to i8*
  %4 = bitcast %struct.point268* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo269(%struct.point269* noalias sret(%struct.point269) align 4 %0, %struct.point269* noundef byval(%struct.point269) align 4 %1) #0 {
  %3 = bitcast %struct.point269* %0 to i8*
  %4 = bitcast %struct.point269* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo270(%struct.point270* noalias sret(%struct.point270) align 4 %0, %struct.point270* noundef byval(%struct.point270) align 4 %1) #0 {
  %3 = bitcast %struct.point270* %0 to i8*
  %4 = bitcast %struct.point270* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo271(%struct.point271* noalias sret(%struct.point271) align 4 %0, %struct.point271* noundef byval(%struct.point271) align 4 %1) #0 {
  %3 = bitcast %struct.point271* %0 to i8*
  %4 = bitcast %struct.point271* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo272(%struct.point272* noalias sret(%struct.point272) align 4 %0, %struct.point272* noundef byval(%struct.point272) align 4 %1) #0 {
  %3 = bitcast %struct.point272* %0 to i8*
  %4 = bitcast %struct.point272* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo273(%struct.point273* noalias sret(%struct.point273) align 4 %0, %struct.point273* noundef byval(%struct.point273) align 4 %1) #0 {
  %3 = bitcast %struct.point273* %0 to i8*
  %4 = bitcast %struct.point273* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo274(%struct.point274* noalias sret(%struct.point274) align 4 %0, %struct.point274* noundef byval(%struct.point274) align 4 %1) #0 {
  %3 = bitcast %struct.point274* %0 to i8*
  %4 = bitcast %struct.point274* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo275(%struct.point275* noalias sret(%struct.point275) align 4 %0, %struct.point275* noundef byval(%struct.point275) align 4 %1) #0 {
  %3 = bitcast %struct.point275* %0 to i8*
  %4 = bitcast %struct.point275* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo276(%struct.point276* noalias sret(%struct.point276) align 4 %0, %struct.point276* noundef byval(%struct.point276) align 4 %1) #0 {
  %3 = bitcast %struct.point276* %0 to i8*
  %4 = bitcast %struct.point276* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo277(%struct.point277* noalias sret(%struct.point277) align 4 %0, %struct.point277* noundef byval(%struct.point277) align 4 %1) #0 {
  %3 = bitcast %struct.point277* %0 to i8*
  %4 = bitcast %struct.point277* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo278(%struct.point278* noalias sret(%struct.point278) align 4 %0, %struct.point278* noundef byval(%struct.point278) align 4 %1) #0 {
  %3 = bitcast %struct.point278* %0 to i8*
  %4 = bitcast %struct.point278* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo279(%struct.point279* noalias sret(%struct.point279) align 4 %0, %struct.point279* noundef byval(%struct.point279) align 4 %1) #0 {
  %3 = bitcast %struct.point279* %0 to i8*
  %4 = bitcast %struct.point279* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo280(%struct.point280* noalias sret(%struct.point280) align 4 %0, %struct.point280* noundef byval(%struct.point280) align 4 %1) #0 {
  %3 = bitcast %struct.point280* %0 to i8*
  %4 = bitcast %struct.point280* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo281(%struct.point281* noalias sret(%struct.point281) align 4 %0, %struct.point281* noundef byval(%struct.point281) align 4 %1) #0 {
  %3 = bitcast %struct.point281* %0 to i8*
  %4 = bitcast %struct.point281* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo282(%struct.point282* noalias sret(%struct.point282) align 4 %0, %struct.point282* noundef byval(%struct.point282) align 4 %1) #0 {
  %3 = bitcast %struct.point282* %0 to i8*
  %4 = bitcast %struct.point282* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo283(%struct.point283* noalias sret(%struct.point283) align 4 %0, %struct.point283* noundef byval(%struct.point283) align 4 %1) #0 {
  %3 = bitcast %struct.point283* %0 to i8*
  %4 = bitcast %struct.point283* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo284(%struct.point284* noalias sret(%struct.point284) align 4 %0, %struct.point284* noundef byval(%struct.point284) align 4 %1) #0 {
  %3 = bitcast %struct.point284* %0 to i8*
  %4 = bitcast %struct.point284* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo285(%struct.point285* noalias sret(%struct.point285) align 4 %0, %struct.point285* noundef byval(%struct.point285) align 4 %1) #0 {
  %3 = bitcast %struct.point285* %0 to i8*
  %4 = bitcast %struct.point285* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo286(%struct.point286* noalias sret(%struct.point286) align 4 %0, %struct.point286* noundef byval(%struct.point286) align 4 %1) #0 {
  %3 = bitcast %struct.point286* %0 to i8*
  %4 = bitcast %struct.point286* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo287(%struct.point287* noalias sret(%struct.point287) align 4 %0, %struct.point287* noundef byval(%struct.point287) align 4 %1) #0 {
  %3 = bitcast %struct.point287* %0 to i8*
  %4 = bitcast %struct.point287* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo288(%struct.point288* noalias sret(%struct.point288) align 4 %0, %struct.point288* noundef byval(%struct.point288) align 4 %1) #0 {
  %3 = bitcast %struct.point288* %0 to i8*
  %4 = bitcast %struct.point288* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo289(%struct.point289* noalias sret(%struct.point289) align 4 %0, %struct.point289* noundef byval(%struct.point289) align 4 %1) #0 {
  %3 = bitcast %struct.point289* %0 to i8*
  %4 = bitcast %struct.point289* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo290(%struct.point290* noalias sret(%struct.point290) align 4 %0, %struct.point290* noundef byval(%struct.point290) align 4 %1) #0 {
  %3 = bitcast %struct.point290* %0 to i8*
  %4 = bitcast %struct.point290* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo291(%struct.point291* noalias sret(%struct.point291) align 4 %0, %struct.point291* noundef byval(%struct.point291) align 4 %1) #0 {
  %3 = bitcast %struct.point291* %0 to i8*
  %4 = bitcast %struct.point291* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo292(%struct.point292* noalias sret(%struct.point292) align 4 %0, %struct.point292* noundef byval(%struct.point292) align 4 %1) #0 {
  %3 = bitcast %struct.point292* %0 to i8*
  %4 = bitcast %struct.point292* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo293(%struct.point293* noalias sret(%struct.point293) align 4 %0, %struct.point293* noundef byval(%struct.point293) align 4 %1) #0 {
  %3 = bitcast %struct.point293* %0 to i8*
  %4 = bitcast %struct.point293* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo294(%struct.point294* noalias sret(%struct.point294) align 4 %0, %struct.point294* noundef byval(%struct.point294) align 4 %1) #0 {
  %3 = bitcast %struct.point294* %0 to i8*
  %4 = bitcast %struct.point294* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo295(%struct.point295* noalias sret(%struct.point295) align 2 %0, %struct.point295* noundef byval(%struct.point295) align 2 %1) #0 {
  %3 = bitcast %struct.point295* %0 to i8*
  %4 = bitcast %struct.point295* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 4, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo296(%struct.point296* noalias sret(%struct.point296) align 2 %0, %struct.point296* noundef byval(%struct.point296) align 2 %1) #0 {
  %3 = bitcast %struct.point296* %0 to i8*
  %4 = bitcast %struct.point296* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 6, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo297(%struct.point297* noalias sret(%struct.point297) align 2 %0, %struct.point297* noundef byval(%struct.point297) align 2 %1) #0 {
  %3 = bitcast %struct.point297* %0 to i8*
  %4 = bitcast %struct.point297* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 6, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo298(%struct.point298* noalias sret(%struct.point298) align 2 %0, %struct.point298* noundef byval(%struct.point298) align 2 %1) #0 {
  %3 = bitcast %struct.point298* %0 to i8*
  %4 = bitcast %struct.point298* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo299(%struct.point299* noalias sret(%struct.point299) align 4 %0, %struct.point299* noundef byval(%struct.point299) align 4 %1) #0 {
  %3 = bitcast %struct.point299* %0 to i8*
  %4 = bitcast %struct.point299* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo300(%struct.point300* noalias sret(%struct.point300) align 4 %0, %struct.point300* noundef byval(%struct.point300) align 4 %1) #0 {
  %3 = bitcast %struct.point300* %0 to i8*
  %4 = bitcast %struct.point300* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo301(%struct.point301* noalias sret(%struct.point301) align 2 %0, %struct.point301* noundef byval(%struct.point301) align 2 %1) #0 {
  %3 = bitcast %struct.point301* %0 to i8*
  %4 = bitcast %struct.point301* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 6, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo302(%struct.point302* noalias sret(%struct.point302) align 2 %0, %struct.point302* noundef byval(%struct.point302) align 2 %1) #0 {
  %3 = bitcast %struct.point302* %0 to i8*
  %4 = bitcast %struct.point302* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo303(%struct.point303* noalias sret(%struct.point303) align 2 %0, %struct.point303* noundef byval(%struct.point303) align 2 %1) #0 {
  %3 = bitcast %struct.point303* %0 to i8*
  %4 = bitcast %struct.point303* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo304(%struct.point304* noalias sret(%struct.point304) align 4 %0, %struct.point304* noundef byval(%struct.point304) align 4 %1) #0 {
  %3 = bitcast %struct.point304* %0 to i8*
  %4 = bitcast %struct.point304* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo305(%struct.point305* noalias sret(%struct.point305) align 4 %0, %struct.point305* noundef byval(%struct.point305) align 4 %1) #0 {
  %3 = bitcast %struct.point305* %0 to i8*
  %4 = bitcast %struct.point305* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo306(%struct.point306* noalias sret(%struct.point306) align 4 %0, %struct.point306* noundef byval(%struct.point306) align 4 %1) #0 {
  %3 = bitcast %struct.point306* %0 to i8*
  %4 = bitcast %struct.point306* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo307(%struct.point307* noalias sret(%struct.point307) align 4 %0, %struct.point307* noundef byval(%struct.point307) align 4 %1) #0 {
  %3 = bitcast %struct.point307* %0 to i8*
  %4 = bitcast %struct.point307* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo308(%struct.point308* noalias sret(%struct.point308) align 4 %0, %struct.point308* noundef byval(%struct.point308) align 4 %1) #0 {
  %3 = bitcast %struct.point308* %0 to i8*
  %4 = bitcast %struct.point308* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo309(%struct.point309* noalias sret(%struct.point309) align 4 %0, %struct.point309* noundef byval(%struct.point309) align 4 %1) #0 {
  %3 = bitcast %struct.point309* %0 to i8*
  %4 = bitcast %struct.point309* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo310(%struct.point310* noalias sret(%struct.point310) align 4 %0, %struct.point310* noundef byval(%struct.point310) align 4 %1) #0 {
  %3 = bitcast %struct.point310* %0 to i8*
  %4 = bitcast %struct.point310* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo311(%struct.point311* noalias sret(%struct.point311) align 4 %0, %struct.point311* noundef byval(%struct.point311) align 4 %1) #0 {
  %3 = bitcast %struct.point311* %0 to i8*
  %4 = bitcast %struct.point311* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo312(%struct.point312* noalias sret(%struct.point312) align 4 %0, %struct.point312* noundef byval(%struct.point312) align 4 %1) #0 {
  %3 = bitcast %struct.point312* %0 to i8*
  %4 = bitcast %struct.point312* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo313(%struct.point313* noalias sret(%struct.point313) align 4 %0, %struct.point313* noundef byval(%struct.point313) align 4 %1) #0 {
  %3 = bitcast %struct.point313* %0 to i8*
  %4 = bitcast %struct.point313* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo314(%struct.point314* noalias sret(%struct.point314) align 4 %0, %struct.point314* noundef byval(%struct.point314) align 4 %1) #0 {
  %3 = bitcast %struct.point314* %0 to i8*
  %4 = bitcast %struct.point314* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo315(%struct.point315* noalias sret(%struct.point315) align 4 %0, %struct.point315* noundef byval(%struct.point315) align 4 %1) #0 {
  %3 = bitcast %struct.point315* %0 to i8*
  %4 = bitcast %struct.point315* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo316(%struct.point316* noalias sret(%struct.point316) align 2 %0, %struct.point316* noundef byval(%struct.point316) align 2 %1) #0 {
  %3 = bitcast %struct.point316* %0 to i8*
  %4 = bitcast %struct.point316* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 6, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo317(%struct.point317* noalias sret(%struct.point317) align 2 %0, %struct.point317* noundef byval(%struct.point317) align 2 %1) #0 {
  %3 = bitcast %struct.point317* %0 to i8*
  %4 = bitcast %struct.point317* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo318(%struct.point318* noalias sret(%struct.point318) align 2 %0, %struct.point318* noundef byval(%struct.point318) align 2 %1) #0 {
  %3 = bitcast %struct.point318* %0 to i8*
  %4 = bitcast %struct.point318* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo319(%struct.point319* noalias sret(%struct.point319) align 2 %0, %struct.point319* noundef byval(%struct.point319) align 2 %1) #0 {
  %3 = bitcast %struct.point319* %0 to i8*
  %4 = bitcast %struct.point319* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo320(%struct.point320* noalias sret(%struct.point320) align 4 %0, %struct.point320* noundef byval(%struct.point320) align 4 %1) #0 {
  %3 = bitcast %struct.point320* %0 to i8*
  %4 = bitcast %struct.point320* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo321(%struct.point321* noalias sret(%struct.point321) align 4 %0, %struct.point321* noundef byval(%struct.point321) align 4 %1) #0 {
  %3 = bitcast %struct.point321* %0 to i8*
  %4 = bitcast %struct.point321* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo322(%struct.point322* noalias sret(%struct.point322) align 2 %0, %struct.point322* noundef byval(%struct.point322) align 2 %1) #0 {
  %3 = bitcast %struct.point322* %0 to i8*
  %4 = bitcast %struct.point322* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo323(%struct.point323* noalias sret(%struct.point323) align 2 %0, %struct.point323* noundef byval(%struct.point323) align 2 %1) #0 {
  %3 = bitcast %struct.point323* %0 to i8*
  %4 = bitcast %struct.point323* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo324(%struct.point324* noalias sret(%struct.point324) align 2 %0, %struct.point324* noundef byval(%struct.point324) align 2 %1) #0 {
  %3 = bitcast %struct.point324* %0 to i8*
  %4 = bitcast %struct.point324* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 2 %3, i8* align 2 %4, i32 10, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo325(%struct.point325* noalias sret(%struct.point325) align 4 %0, %struct.point325* noundef byval(%struct.point325) align 4 %1) #0 {
  %3 = bitcast %struct.point325* %0 to i8*
  %4 = bitcast %struct.point325* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo326(%struct.point326* noalias sret(%struct.point326) align 4 %0, %struct.point326* noundef byval(%struct.point326) align 4 %1) #0 {
  %3 = bitcast %struct.point326* %0 to i8*
  %4 = bitcast %struct.point326* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo327(%struct.point327* noalias sret(%struct.point327) align 4 %0, %struct.point327* noundef byval(%struct.point327) align 4 %1) #0 {
  %3 = bitcast %struct.point327* %0 to i8*
  %4 = bitcast %struct.point327* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo328(%struct.point328* noalias sret(%struct.point328) align 4 %0, %struct.point328* noundef byval(%struct.point328) align 4 %1) #0 {
  %3 = bitcast %struct.point328* %0 to i8*
  %4 = bitcast %struct.point328* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo329(%struct.point329* noalias sret(%struct.point329) align 4 %0, %struct.point329* noundef byval(%struct.point329) align 4 %1) #0 {
  %3 = bitcast %struct.point329* %0 to i8*
  %4 = bitcast %struct.point329* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo330(%struct.point330* noalias sret(%struct.point330) align 4 %0, %struct.point330* noundef byval(%struct.point330) align 4 %1) #0 {
  %3 = bitcast %struct.point330* %0 to i8*
  %4 = bitcast %struct.point330* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo331(%struct.point331* noalias sret(%struct.point331) align 4 %0, %struct.point331* noundef byval(%struct.point331) align 4 %1) #0 {
  %3 = bitcast %struct.point331* %0 to i8*
  %4 = bitcast %struct.point331* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo332(%struct.point332* noalias sret(%struct.point332) align 4 %0, %struct.point332* noundef byval(%struct.point332) align 4 %1) #0 {
  %3 = bitcast %struct.point332* %0 to i8*
  %4 = bitcast %struct.point332* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo333(%struct.point333* noalias sret(%struct.point333) align 4 %0, %struct.point333* noundef byval(%struct.point333) align 4 %1) #0 {
  %3 = bitcast %struct.point333* %0 to i8*
  %4 = bitcast %struct.point333* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo334(%struct.point334* noalias sret(%struct.point334) align 4 %0, %struct.point334* noundef byval(%struct.point334) align 4 %1) #0 {
  %3 = bitcast %struct.point334* %0 to i8*
  %4 = bitcast %struct.point334* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo335(%struct.point335* noalias sret(%struct.point335) align 4 %0, %struct.point335* noundef byval(%struct.point335) align 4 %1) #0 {
  %3 = bitcast %struct.point335* %0 to i8*
  %4 = bitcast %struct.point335* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo336(%struct.point336* noalias sret(%struct.point336) align 4 %0, %struct.point336* noundef byval(%struct.point336) align 4 %1) #0 {
  %3 = bitcast %struct.point336* %0 to i8*
  %4 = bitcast %struct.point336* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo337(%struct.point337* noalias sret(%struct.point337) align 4 %0, %struct.point337* noundef byval(%struct.point337) align 4 %1) #0 {
  %3 = bitcast %struct.point337* %0 to i8*
  %4 = bitcast %struct.point337* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo338(%struct.point338* noalias sret(%struct.point338) align 4 %0, %struct.point338* noundef byval(%struct.point338) align 4 %1) #0 {
  %3 = bitcast %struct.point338* %0 to i8*
  %4 = bitcast %struct.point338* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo339(%struct.point339* noalias sret(%struct.point339) align 4 %0, %struct.point339* noundef byval(%struct.point339) align 4 %1) #0 {
  %3 = bitcast %struct.point339* %0 to i8*
  %4 = bitcast %struct.point339* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo340(%struct.point340* noalias sret(%struct.point340) align 4 %0, %struct.point340* noundef byval(%struct.point340) align 4 %1) #0 {
  %3 = bitcast %struct.point340* %0 to i8*
  %4 = bitcast %struct.point340* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo341(%struct.point341* noalias sret(%struct.point341) align 4 %0, %struct.point341* noundef byval(%struct.point341) align 4 %1) #0 {
  %3 = bitcast %struct.point341* %0 to i8*
  %4 = bitcast %struct.point341* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo342(%struct.point342* noalias sret(%struct.point342) align 4 %0, %struct.point342* noundef byval(%struct.point342) align 4 %1) #0 {
  %3 = bitcast %struct.point342* %0 to i8*
  %4 = bitcast %struct.point342* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo343(%struct.point343* noalias sret(%struct.point343) align 4 %0, %struct.point343* noundef byval(%struct.point343) align 4 %1) #0 {
  %3 = bitcast %struct.point343* %0 to i8*
  %4 = bitcast %struct.point343* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo344(%struct.point344* noalias sret(%struct.point344) align 4 %0, %struct.point344* noundef byval(%struct.point344) align 4 %1) #0 {
  %3 = bitcast %struct.point344* %0 to i8*
  %4 = bitcast %struct.point344* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo345(%struct.point345* noalias sret(%struct.point345) align 4 %0, %struct.point345* noundef byval(%struct.point345) align 4 %1) #0 {
  %3 = bitcast %struct.point345* %0 to i8*
  %4 = bitcast %struct.point345* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo346(%struct.point346* noalias sret(%struct.point346) align 4 %0, %struct.point346* noundef byval(%struct.point346) align 4 %1) #0 {
  %3 = bitcast %struct.point346* %0 to i8*
  %4 = bitcast %struct.point346* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo347(%struct.point347* noalias sret(%struct.point347) align 4 %0, %struct.point347* noundef byval(%struct.point347) align 4 %1) #0 {
  %3 = bitcast %struct.point347* %0 to i8*
  %4 = bitcast %struct.point347* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo348(%struct.point348* noalias sret(%struct.point348) align 4 %0, %struct.point348* noundef byval(%struct.point348) align 4 %1) #0 {
  %3 = bitcast %struct.point348* %0 to i8*
  %4 = bitcast %struct.point348* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo349(%struct.point349* noalias sret(%struct.point349) align 4 %0, %struct.point349* noundef byval(%struct.point349) align 4 %1) #0 {
  %3 = bitcast %struct.point349* %0 to i8*
  %4 = bitcast %struct.point349* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo350(%struct.point350* noalias sret(%struct.point350) align 4 %0, %struct.point350* noundef byval(%struct.point350) align 4 %1) #0 {
  %3 = bitcast %struct.point350* %0 to i8*
  %4 = bitcast %struct.point350* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo351(%struct.point351* noalias sret(%struct.point351) align 4 %0, %struct.point351* noundef byval(%struct.point351) align 4 %1) #0 {
  %3 = bitcast %struct.point351* %0 to i8*
  %4 = bitcast %struct.point351* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo352(%struct.point352* noalias sret(%struct.point352) align 4 %0, %struct.point352* noundef byval(%struct.point352) align 4 %1) #0 {
  %3 = bitcast %struct.point352* %0 to i8*
  %4 = bitcast %struct.point352* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo353(%struct.point353* noalias sret(%struct.point353) align 4 %0, %struct.point353* noundef byval(%struct.point353) align 4 %1) #0 {
  %3 = bitcast %struct.point353* %0 to i8*
  %4 = bitcast %struct.point353* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo354(%struct.point354* noalias sret(%struct.point354) align 4 %0, %struct.point354* noundef byval(%struct.point354) align 4 %1) #0 {
  %3 = bitcast %struct.point354* %0 to i8*
  %4 = bitcast %struct.point354* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo355(%struct.point355* noalias sret(%struct.point355) align 4 %0, %struct.point355* noundef byval(%struct.point355) align 4 %1) #0 {
  %3 = bitcast %struct.point355* %0 to i8*
  %4 = bitcast %struct.point355* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo356(%struct.point356* noalias sret(%struct.point356) align 4 %0, %struct.point356* noundef byval(%struct.point356) align 4 %1) #0 {
  %3 = bitcast %struct.point356* %0 to i8*
  %4 = bitcast %struct.point356* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo357(%struct.point357* noalias sret(%struct.point357) align 4 %0, %struct.point357* noundef byval(%struct.point357) align 4 %1) #0 {
  %3 = bitcast %struct.point357* %0 to i8*
  %4 = bitcast %struct.point357* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo358(%struct.point358* noalias sret(%struct.point358) align 4 %0, %struct.point358* noundef byval(%struct.point358) align 4 %1) #0 {
  %3 = bitcast %struct.point358* %0 to i8*
  %4 = bitcast %struct.point358* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo359(%struct.point359* noalias sret(%struct.point359) align 4 %0, %struct.point359* noundef byval(%struct.point359) align 4 %1) #0 {
  %3 = bitcast %struct.point359* %0 to i8*
  %4 = bitcast %struct.point359* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo360(%struct.point360* noalias sret(%struct.point360) align 4 %0, %struct.point360* noundef byval(%struct.point360) align 4 %1) #0 {
  %3 = bitcast %struct.point360* %0 to i8*
  %4 = bitcast %struct.point360* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo361(%struct.point361* noalias sret(%struct.point361) align 4 %0, %struct.point361* noundef byval(%struct.point361) align 4 %1) #0 {
  %3 = bitcast %struct.point361* %0 to i8*
  %4 = bitcast %struct.point361* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo362(%struct.point362* noalias sret(%struct.point362) align 4 %0, %struct.point362* noundef byval(%struct.point362) align 4 %1) #0 {
  %3 = bitcast %struct.point362* %0 to i8*
  %4 = bitcast %struct.point362* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo363(%struct.point363* noalias sret(%struct.point363) align 4 %0, %struct.point363* noundef byval(%struct.point363) align 4 %1) #0 {
  %3 = bitcast %struct.point363* %0 to i8*
  %4 = bitcast %struct.point363* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo364(%struct.point364* noalias sret(%struct.point364) align 4 %0, %struct.point364* noundef byval(%struct.point364) align 4 %1) #0 {
  %3 = bitcast %struct.point364* %0 to i8*
  %4 = bitcast %struct.point364* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo365(%struct.point365* noalias sret(%struct.point365) align 4 %0, %struct.point365* noundef byval(%struct.point365) align 4 %1) #0 {
  %3 = bitcast %struct.point365* %0 to i8*
  %4 = bitcast %struct.point365* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo366(%struct.point366* noalias sret(%struct.point366) align 4 %0, %struct.point366* noundef byval(%struct.point366) align 4 %1) #0 {
  %3 = bitcast %struct.point366* %0 to i8*
  %4 = bitcast %struct.point366* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo367(%struct.point367* noalias sret(%struct.point367) align 4 %0, %struct.point367* noundef byval(%struct.point367) align 4 %1) #0 {
  %3 = bitcast %struct.point367* %0 to i8*
  %4 = bitcast %struct.point367* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo368(%struct.point368* noalias sret(%struct.point368) align 4 %0, %struct.point368* noundef byval(%struct.point368) align 4 %1) #0 {
  %3 = bitcast %struct.point368* %0 to i8*
  %4 = bitcast %struct.point368* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo369(%struct.point369* noalias sret(%struct.point369) align 4 %0, %struct.point369* noundef byval(%struct.point369) align 4 %1) #0 {
  %3 = bitcast %struct.point369* %0 to i8*
  %4 = bitcast %struct.point369* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo370(%struct.point370* noalias sret(%struct.point370) align 4 %0, %struct.point370* noundef byval(%struct.point370) align 4 %1) #0 {
  %3 = bitcast %struct.point370* %0 to i8*
  %4 = bitcast %struct.point370* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo371(%struct.point371* noalias sret(%struct.point371) align 4 %0, %struct.point371* noundef byval(%struct.point371) align 4 %1) #0 {
  %3 = bitcast %struct.point371* %0 to i8*
  %4 = bitcast %struct.point371* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo372(%struct.point372* noalias sret(%struct.point372) align 4 %0, %struct.point372* noundef byval(%struct.point372) align 4 %1) #0 {
  %3 = bitcast %struct.point372* %0 to i8*
  %4 = bitcast %struct.point372* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo373(%struct.point373* noalias sret(%struct.point373) align 4 %0, %struct.point373* noundef byval(%struct.point373) align 4 %1) #0 {
  %3 = bitcast %struct.point373* %0 to i8*
  %4 = bitcast %struct.point373* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo374(%struct.point374* noalias sret(%struct.point374) align 4 %0, %struct.point374* noundef byval(%struct.point374) align 4 %1) #0 {
  %3 = bitcast %struct.point374* %0 to i8*
  %4 = bitcast %struct.point374* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo375(%struct.point375* noalias sret(%struct.point375) align 4 %0, %struct.point375* noundef byval(%struct.point375) align 4 %1) #0 {
  %3 = bitcast %struct.point375* %0 to i8*
  %4 = bitcast %struct.point375* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo376(%struct.point376* noalias sret(%struct.point376) align 4 %0, %struct.point376* noundef byval(%struct.point376) align 4 %1) #0 {
  %3 = bitcast %struct.point376* %0 to i8*
  %4 = bitcast %struct.point376* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo377(%struct.point377* noalias sret(%struct.point377) align 4 %0, %struct.point377* noundef byval(%struct.point377) align 4 %1) #0 {
  %3 = bitcast %struct.point377* %0 to i8*
  %4 = bitcast %struct.point377* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo378(%struct.point378* noalias sret(%struct.point378) align 4 %0, %struct.point378* noundef byval(%struct.point378) align 4 %1) #0 {
  %3 = bitcast %struct.point378* %0 to i8*
  %4 = bitcast %struct.point378* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo379(%struct.point379* noalias sret(%struct.point379) align 4 %0, %struct.point379* noundef byval(%struct.point379) align 4 %1) #0 {
  %3 = bitcast %struct.point379* %0 to i8*
  %4 = bitcast %struct.point379* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo380(%struct.point380* noalias sret(%struct.point380) align 4 %0, %struct.point380* noundef byval(%struct.point380) align 4 %1) #0 {
  %3 = bitcast %struct.point380* %0 to i8*
  %4 = bitcast %struct.point380* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo381(%struct.point381* noalias sret(%struct.point381) align 4 %0, %struct.point381* noundef byval(%struct.point381) align 4 %1) #0 {
  %3 = bitcast %struct.point381* %0 to i8*
  %4 = bitcast %struct.point381* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo382(%struct.point382* noalias sret(%struct.point382) align 4 %0, %struct.point382* noundef byval(%struct.point382) align 4 %1) #0 {
  %3 = bitcast %struct.point382* %0 to i8*
  %4 = bitcast %struct.point382* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo383(%struct.point383* noalias sret(%struct.point383) align 4 %0, %struct.point383* noundef byval(%struct.point383) align 4 %1) #0 {
  %3 = bitcast %struct.point383* %0 to i8*
  %4 = bitcast %struct.point383* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo384(%struct.point384* noalias sret(%struct.point384) align 4 %0, %struct.point384* noundef byval(%struct.point384) align 4 %1) #0 {
  %3 = bitcast %struct.point384* %0 to i8*
  %4 = bitcast %struct.point384* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo385(%struct.point385* noalias sret(%struct.point385) align 4 %0, %struct.point385* noundef byval(%struct.point385) align 4 %1) #0 {
  %3 = bitcast %struct.point385* %0 to i8*
  %4 = bitcast %struct.point385* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo386(%struct.point386* noalias sret(%struct.point386) align 4 %0, %struct.point386* noundef byval(%struct.point386) align 4 %1) #0 {
  %3 = bitcast %struct.point386* %0 to i8*
  %4 = bitcast %struct.point386* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo387(%struct.point387* noalias sret(%struct.point387) align 4 %0, %struct.point387* noundef byval(%struct.point387) align 4 %1) #0 {
  %3 = bitcast %struct.point387* %0 to i8*
  %4 = bitcast %struct.point387* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo388(%struct.point388* noalias sret(%struct.point388) align 4 %0, %struct.point388* noundef byval(%struct.point388) align 4 %1) #0 {
  %3 = bitcast %struct.point388* %0 to i8*
  %4 = bitcast %struct.point388* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo389(%struct.point389* noalias sret(%struct.point389) align 4 %0, %struct.point389* noundef byval(%struct.point389) align 4 %1) #0 {
  %3 = bitcast %struct.point389* %0 to i8*
  %4 = bitcast %struct.point389* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo390(%struct.point390* noalias sret(%struct.point390) align 4 %0, %struct.point390* noundef byval(%struct.point390) align 4 %1) #0 {
  %3 = bitcast %struct.point390* %0 to i8*
  %4 = bitcast %struct.point390* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo391(%struct.point391* noalias sret(%struct.point391) align 4 %0, %struct.point391* noundef byval(%struct.point391) align 4 %1) #0 {
  %3 = bitcast %struct.point391* %0 to i8*
  %4 = bitcast %struct.point391* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo392(%struct.point392* noalias sret(%struct.point392) align 4 %0, %struct.point392* noundef byval(%struct.point392) align 4 %1) #0 {
  %3 = bitcast %struct.point392* %0 to i8*
  %4 = bitcast %struct.point392* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo393(%struct.point393* noalias sret(%struct.point393) align 4 %0, %struct.point393* noundef byval(%struct.point393) align 4 %1) #0 {
  %3 = bitcast %struct.point393* %0 to i8*
  %4 = bitcast %struct.point393* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo394(%struct.point394* noalias sret(%struct.point394) align 4 %0, %struct.point394* noundef byval(%struct.point394) align 4 %1) #0 {
  %3 = bitcast %struct.point394* %0 to i8*
  %4 = bitcast %struct.point394* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo395(%struct.point395* noalias sret(%struct.point395) align 4 %0, %struct.point395* noundef byval(%struct.point395) align 4 %1) #0 {
  %3 = bitcast %struct.point395* %0 to i8*
  %4 = bitcast %struct.point395* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo396(%struct.point396* noalias sret(%struct.point396) align 4 %0, %struct.point396* noundef byval(%struct.point396) align 4 %1) #0 {
  %3 = bitcast %struct.point396* %0 to i8*
  %4 = bitcast %struct.point396* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo397(%struct.point397* noalias sret(%struct.point397) align 4 %0, %struct.point397* noundef byval(%struct.point397) align 4 %1) #0 {
  %3 = bitcast %struct.point397* %0 to i8*
  %4 = bitcast %struct.point397* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo398(%struct.point398* noalias sret(%struct.point398) align 4 %0, %struct.point398* noundef byval(%struct.point398) align 4 %1) #0 {
  %3 = bitcast %struct.point398* %0 to i8*
  %4 = bitcast %struct.point398* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo399(%struct.point399* noalias sret(%struct.point399) align 4 %0, %struct.point399* noundef byval(%struct.point399) align 4 %1) #0 {
  %3 = bitcast %struct.point399* %0 to i8*
  %4 = bitcast %struct.point399* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo400(%struct.point400* noalias sret(%struct.point400) align 4 %0, %struct.point400* noundef byval(%struct.point400) align 4 %1) #0 {
  %3 = bitcast %struct.point400* %0 to i8*
  %4 = bitcast %struct.point400* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo401(%struct.point401* noalias sret(%struct.point401) align 4 %0, %struct.point401* noundef byval(%struct.point401) align 4 %1) #0 {
  %3 = bitcast %struct.point401* %0 to i8*
  %4 = bitcast %struct.point401* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo402(%struct.point402* noalias sret(%struct.point402) align 4 %0, %struct.point402* noundef byval(%struct.point402) align 4 %1) #0 {
  %3 = bitcast %struct.point402* %0 to i8*
  %4 = bitcast %struct.point402* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo403(%struct.point403* noalias sret(%struct.point403) align 4 %0, %struct.point403* noundef byval(%struct.point403) align 4 %1) #0 {
  %3 = bitcast %struct.point403* %0 to i8*
  %4 = bitcast %struct.point403* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo404(%struct.point404* noalias sret(%struct.point404) align 4 %0, %struct.point404* noundef byval(%struct.point404) align 4 %1) #0 {
  %3 = bitcast %struct.point404* %0 to i8*
  %4 = bitcast %struct.point404* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo405(%struct.point405* noalias sret(%struct.point405) align 4 %0, %struct.point405* noundef byval(%struct.point405) align 4 %1) #0 {
  %3 = bitcast %struct.point405* %0 to i8*
  %4 = bitcast %struct.point405* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo406(%struct.point406* noalias sret(%struct.point406) align 4 %0, %struct.point406* noundef byval(%struct.point406) align 4 %1) #0 {
  %3 = bitcast %struct.point406* %0 to i8*
  %4 = bitcast %struct.point406* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo407(%struct.point407* noalias sret(%struct.point407) align 4 %0, %struct.point407* noundef byval(%struct.point407) align 4 %1) #0 {
  %3 = bitcast %struct.point407* %0 to i8*
  %4 = bitcast %struct.point407* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo408(%struct.point408* noalias sret(%struct.point408) align 4 %0, %struct.point408* noundef byval(%struct.point408) align 4 %1) #0 {
  %3 = bitcast %struct.point408* %0 to i8*
  %4 = bitcast %struct.point408* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo409(%struct.point409* noalias sret(%struct.point409) align 4 %0, %struct.point409* noundef byval(%struct.point409) align 4 %1) #0 {
  %3 = bitcast %struct.point409* %0 to i8*
  %4 = bitcast %struct.point409* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo410(%struct.point410* noalias sret(%struct.point410) align 4 %0, %struct.point410* noundef byval(%struct.point410) align 4 %1) #0 {
  %3 = bitcast %struct.point410* %0 to i8*
  %4 = bitcast %struct.point410* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo411(%struct.point411* noalias sret(%struct.point411) align 4 %0, %struct.point411* noundef byval(%struct.point411) align 4 %1) #0 {
  %3 = bitcast %struct.point411* %0 to i8*
  %4 = bitcast %struct.point411* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo412(%struct.point412* noalias sret(%struct.point412) align 4 %0, %struct.point412* noundef byval(%struct.point412) align 4 %1) #0 {
  %3 = bitcast %struct.point412* %0 to i8*
  %4 = bitcast %struct.point412* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo413(%struct.point413* noalias sret(%struct.point413) align 4 %0, %struct.point413* noundef byval(%struct.point413) align 4 %1) #0 {
  %3 = bitcast %struct.point413* %0 to i8*
  %4 = bitcast %struct.point413* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo414(%struct.point414* noalias sret(%struct.point414) align 4 %0, %struct.point414* noundef byval(%struct.point414) align 4 %1) #0 {
  %3 = bitcast %struct.point414* %0 to i8*
  %4 = bitcast %struct.point414* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo415(%struct.point415* noalias sret(%struct.point415) align 4 %0, %struct.point415* noundef byval(%struct.point415) align 4 %1) #0 {
  %3 = bitcast %struct.point415* %0 to i8*
  %4 = bitcast %struct.point415* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo416(%struct.point416* noalias sret(%struct.point416) align 4 %0, %struct.point416* noundef byval(%struct.point416) align 4 %1) #0 {
  %3 = bitcast %struct.point416* %0 to i8*
  %4 = bitcast %struct.point416* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo417(%struct.point417* noalias sret(%struct.point417) align 4 %0, %struct.point417* noundef byval(%struct.point417) align 4 %1) #0 {
  %3 = bitcast %struct.point417* %0 to i8*
  %4 = bitcast %struct.point417* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo418(%struct.point418* noalias sret(%struct.point418) align 4 %0, %struct.point418* noundef byval(%struct.point418) align 4 %1) #0 {
  %3 = bitcast %struct.point418* %0 to i8*
  %4 = bitcast %struct.point418* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo419(%struct.point419* noalias sret(%struct.point419) align 4 %0, %struct.point419* noundef byval(%struct.point419) align 4 %1) #0 {
  %3 = bitcast %struct.point419* %0 to i8*
  %4 = bitcast %struct.point419* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo420(%struct.point420* noalias sret(%struct.point420) align 4 %0, %struct.point420* noundef byval(%struct.point420) align 4 %1) #0 {
  %3 = bitcast %struct.point420* %0 to i8*
  %4 = bitcast %struct.point420* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo421(%struct.point421* noalias sret(%struct.point421) align 4 %0, %struct.point421* noundef byval(%struct.point421) align 4 %1) #0 {
  %3 = bitcast %struct.point421* %0 to i8*
  %4 = bitcast %struct.point421* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo422(%struct.point422* noalias sret(%struct.point422) align 4 %0, %struct.point422* noundef byval(%struct.point422) align 4 %1) #0 {
  %3 = bitcast %struct.point422* %0 to i8*
  %4 = bitcast %struct.point422* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo423(%struct.point423* noalias sret(%struct.point423) align 4 %0, %struct.point423* noundef byval(%struct.point423) align 4 %1) #0 {
  %3 = bitcast %struct.point423* %0 to i8*
  %4 = bitcast %struct.point423* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo424(%struct.point424* noalias sret(%struct.point424) align 4 %0, %struct.point424* noundef byval(%struct.point424) align 4 %1) #0 {
  %3 = bitcast %struct.point424* %0 to i8*
  %4 = bitcast %struct.point424* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo425(%struct.point425* noalias sret(%struct.point425) align 4 %0, %struct.point425* noundef byval(%struct.point425) align 4 %1) #0 {
  %3 = bitcast %struct.point425* %0 to i8*
  %4 = bitcast %struct.point425* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo426(%struct.point426* noalias sret(%struct.point426) align 4 %0, %struct.point426* noundef byval(%struct.point426) align 4 %1) #0 {
  %3 = bitcast %struct.point426* %0 to i8*
  %4 = bitcast %struct.point426* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo427(%struct.point427* noalias sret(%struct.point427) align 4 %0, %struct.point427* noundef byval(%struct.point427) align 4 %1) #0 {
  %3 = bitcast %struct.point427* %0 to i8*
  %4 = bitcast %struct.point427* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo428(%struct.point428* noalias sret(%struct.point428) align 4 %0, %struct.point428* noundef byval(%struct.point428) align 4 %1) #0 {
  %3 = bitcast %struct.point428* %0 to i8*
  %4 = bitcast %struct.point428* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo429(%struct.point429* noalias sret(%struct.point429) align 4 %0, %struct.point429* noundef byval(%struct.point429) align 4 %1) #0 {
  %3 = bitcast %struct.point429* %0 to i8*
  %4 = bitcast %struct.point429* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo430(%struct.point430* noalias sret(%struct.point430) align 4 %0, %struct.point430* noundef byval(%struct.point430) align 4 %1) #0 {
  %3 = bitcast %struct.point430* %0 to i8*
  %4 = bitcast %struct.point430* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo431(%struct.point431* noalias sret(%struct.point431) align 4 %0, %struct.point431* noundef byval(%struct.point431) align 4 %1) #0 {
  %3 = bitcast %struct.point431* %0 to i8*
  %4 = bitcast %struct.point431* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo432(%struct.point432* noalias sret(%struct.point432) align 4 %0, %struct.point432* noundef byval(%struct.point432) align 4 %1) #0 {
  %3 = bitcast %struct.point432* %0 to i8*
  %4 = bitcast %struct.point432* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo433(%struct.point433* noalias sret(%struct.point433) align 4 %0, %struct.point433* noundef byval(%struct.point433) align 4 %1) #0 {
  %3 = bitcast %struct.point433* %0 to i8*
  %4 = bitcast %struct.point433* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo434(%struct.point434* noalias sret(%struct.point434) align 4 %0, %struct.point434* noundef byval(%struct.point434) align 4 %1) #0 {
  %3 = bitcast %struct.point434* %0 to i8*
  %4 = bitcast %struct.point434* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo435(%struct.point435* noalias sret(%struct.point435) align 4 %0, %struct.point435* noundef byval(%struct.point435) align 4 %1) #0 {
  %3 = bitcast %struct.point435* %0 to i8*
  %4 = bitcast %struct.point435* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo436(%struct.point436* noalias sret(%struct.point436) align 4 %0, %struct.point436* noundef byval(%struct.point436) align 4 %1) #0 {
  %3 = bitcast %struct.point436* %0 to i8*
  %4 = bitcast %struct.point436* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo437(%struct.point437* noalias sret(%struct.point437) align 4 %0, %struct.point437* noundef byval(%struct.point437) align 4 %1) #0 {
  %3 = bitcast %struct.point437* %0 to i8*
  %4 = bitcast %struct.point437* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo438(%struct.point438* noalias sret(%struct.point438) align 4 %0, %struct.point438* noundef byval(%struct.point438) align 4 %1) #0 {
  %3 = bitcast %struct.point438* %0 to i8*
  %4 = bitcast %struct.point438* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo439(%struct.point439* noalias sret(%struct.point439) align 4 %0, %struct.point439* noundef byval(%struct.point439) align 4 %1) #0 {
  %3 = bitcast %struct.point439* %0 to i8*
  %4 = bitcast %struct.point439* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo440(%struct.point440* noalias sret(%struct.point440) align 4 %0, %struct.point440* noundef byval(%struct.point440) align 4 %1) #0 {
  %3 = bitcast %struct.point440* %0 to i8*
  %4 = bitcast %struct.point440* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo441(%struct.point441* noalias sret(%struct.point441) align 4 %0, %struct.point441* noundef byval(%struct.point441) align 4 %1) #0 {
  %3 = bitcast %struct.point441* %0 to i8*
  %4 = bitcast %struct.point441* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo442(%struct.point442* noalias sret(%struct.point442) align 4 %0, %struct.point442* noundef byval(%struct.point442) align 4 %1) #0 {
  %3 = bitcast %struct.point442* %0 to i8*
  %4 = bitcast %struct.point442* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo443(%struct.point443* noalias sret(%struct.point443) align 4 %0, %struct.point443* noundef byval(%struct.point443) align 4 %1) #0 {
  %3 = bitcast %struct.point443* %0 to i8*
  %4 = bitcast %struct.point443* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo444(%struct.point444* noalias sret(%struct.point444) align 4 %0, %struct.point444* noundef byval(%struct.point444) align 4 %1) #0 {
  %3 = bitcast %struct.point444* %0 to i8*
  %4 = bitcast %struct.point444* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo445(%struct.point445* noalias sret(%struct.point445) align 4 %0, %struct.point445* noundef byval(%struct.point445) align 4 %1) #0 {
  %3 = bitcast %struct.point445* %0 to i8*
  %4 = bitcast %struct.point445* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo446(%struct.point446* noalias sret(%struct.point446) align 4 %0, %struct.point446* noundef byval(%struct.point446) align 4 %1) #0 {
  %3 = bitcast %struct.point446* %0 to i8*
  %4 = bitcast %struct.point446* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo447(%struct.point447* noalias sret(%struct.point447) align 4 %0, %struct.point447* noundef byval(%struct.point447) align 4 %1) #0 {
  %3 = bitcast %struct.point447* %0 to i8*
  %4 = bitcast %struct.point447* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo448(%struct.point448* noalias sret(%struct.point448) align 4 %0, %struct.point448* noundef byval(%struct.point448) align 4 %1) #0 {
  %3 = bitcast %struct.point448* %0 to i8*
  %4 = bitcast %struct.point448* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo449(%struct.point449* noalias sret(%struct.point449) align 4 %0, %struct.point449* noundef byval(%struct.point449) align 4 %1) #0 {
  %3 = bitcast %struct.point449* %0 to i8*
  %4 = bitcast %struct.point449* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo450(%struct.point450* noalias sret(%struct.point450) align 4 %0, %struct.point450* noundef byval(%struct.point450) align 4 %1) #0 {
  %3 = bitcast %struct.point450* %0 to i8*
  %4 = bitcast %struct.point450* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo451(%struct.point451* noalias sret(%struct.point451) align 4 %0, %struct.point451* noundef byval(%struct.point451) align 4 %1) #0 {
  %3 = bitcast %struct.point451* %0 to i8*
  %4 = bitcast %struct.point451* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo452(%struct.point452* noalias sret(%struct.point452) align 4 %0, %struct.point452* noundef byval(%struct.point452) align 4 %1) #0 {
  %3 = bitcast %struct.point452* %0 to i8*
  %4 = bitcast %struct.point452* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo453(%struct.point453* noalias sret(%struct.point453) align 4 %0, %struct.point453* noundef byval(%struct.point453) align 4 %1) #0 {
  %3 = bitcast %struct.point453* %0 to i8*
  %4 = bitcast %struct.point453* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo454(%struct.point454* noalias sret(%struct.point454) align 4 %0, %struct.point454* noundef byval(%struct.point454) align 4 %1) #0 {
  %3 = bitcast %struct.point454* %0 to i8*
  %4 = bitcast %struct.point454* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo455(%struct.point455* noalias sret(%struct.point455) align 4 %0, %struct.point455* noundef byval(%struct.point455) align 4 %1) #0 {
  %3 = bitcast %struct.point455* %0 to i8*
  %4 = bitcast %struct.point455* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo456(%struct.point456* noalias sret(%struct.point456) align 4 %0, %struct.point456* noundef byval(%struct.point456) align 4 %1) #0 {
  %3 = bitcast %struct.point456* %0 to i8*
  %4 = bitcast %struct.point456* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo457(%struct.point457* noalias sret(%struct.point457) align 4 %0, %struct.point457* noundef byval(%struct.point457) align 4 %1) #0 {
  %3 = bitcast %struct.point457* %0 to i8*
  %4 = bitcast %struct.point457* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo458(%struct.point458* noalias sret(%struct.point458) align 4 %0, %struct.point458* noundef byval(%struct.point458) align 4 %1) #0 {
  %3 = bitcast %struct.point458* %0 to i8*
  %4 = bitcast %struct.point458* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo459(%struct.point459* noalias sret(%struct.point459) align 4 %0, %struct.point459* noundef byval(%struct.point459) align 4 %1) #0 {
  %3 = bitcast %struct.point459* %0 to i8*
  %4 = bitcast %struct.point459* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo460(%struct.point460* noalias sret(%struct.point460) align 4 %0, %struct.point460* noundef byval(%struct.point460) align 4 %1) #0 {
  %3 = bitcast %struct.point460* %0 to i8*
  %4 = bitcast %struct.point460* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo461(%struct.point461* noalias sret(%struct.point461) align 4 %0, %struct.point461* noundef byval(%struct.point461) align 4 %1) #0 {
  %3 = bitcast %struct.point461* %0 to i8*
  %4 = bitcast %struct.point461* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo462(%struct.point462* noalias sret(%struct.point462) align 4 %0, %struct.point462* noundef byval(%struct.point462) align 4 %1) #0 {
  %3 = bitcast %struct.point462* %0 to i8*
  %4 = bitcast %struct.point462* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo463(%struct.point463* noalias sret(%struct.point463) align 4 %0, %struct.point463* noundef byval(%struct.point463) align 4 %1) #0 {
  %3 = bitcast %struct.point463* %0 to i8*
  %4 = bitcast %struct.point463* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo464(%struct.point464* noalias sret(%struct.point464) align 4 %0, %struct.point464* noundef byval(%struct.point464) align 4 %1) #0 {
  %3 = bitcast %struct.point464* %0 to i8*
  %4 = bitcast %struct.point464* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo465(%struct.point465* noalias sret(%struct.point465) align 4 %0, %struct.point465* noundef byval(%struct.point465) align 4 %1) #0 {
  %3 = bitcast %struct.point465* %0 to i8*
  %4 = bitcast %struct.point465* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo466(%struct.point466* noalias sret(%struct.point466) align 4 %0, %struct.point466* noundef byval(%struct.point466) align 4 %1) #0 {
  %3 = bitcast %struct.point466* %0 to i8*
  %4 = bitcast %struct.point466* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo467(%struct.point467* noalias sret(%struct.point467) align 4 %0, %struct.point467* noundef byval(%struct.point467) align 4 %1) #0 {
  %3 = bitcast %struct.point467* %0 to i8*
  %4 = bitcast %struct.point467* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo468(%struct.point468* noalias sret(%struct.point468) align 4 %0, %struct.point468* noundef byval(%struct.point468) align 4 %1) #0 {
  %3 = bitcast %struct.point468* %0 to i8*
  %4 = bitcast %struct.point468* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo469(%struct.point469* noalias sret(%struct.point469) align 4 %0, %struct.point469* noundef byval(%struct.point469) align 4 %1) #0 {
  %3 = bitcast %struct.point469* %0 to i8*
  %4 = bitcast %struct.point469* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo470(%struct.point470* noalias sret(%struct.point470) align 4 %0, %struct.point470* noundef byval(%struct.point470) align 4 %1) #0 {
  %3 = bitcast %struct.point470* %0 to i8*
  %4 = bitcast %struct.point470* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo471(%struct.point471* noalias sret(%struct.point471) align 4 %0, %struct.point471* noundef byval(%struct.point471) align 4 %1) #0 {
  %3 = bitcast %struct.point471* %0 to i8*
  %4 = bitcast %struct.point471* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo472(%struct.point472* noalias sret(%struct.point472) align 4 %0, %struct.point472* noundef byval(%struct.point472) align 4 %1) #0 {
  %3 = bitcast %struct.point472* %0 to i8*
  %4 = bitcast %struct.point472* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo473(%struct.point473* noalias sret(%struct.point473) align 4 %0, %struct.point473* noundef byval(%struct.point473) align 4 %1) #0 {
  %3 = bitcast %struct.point473* %0 to i8*
  %4 = bitcast %struct.point473* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo474(%struct.point474* noalias sret(%struct.point474) align 4 %0, %struct.point474* noundef byval(%struct.point474) align 4 %1) #0 {
  %3 = bitcast %struct.point474* %0 to i8*
  %4 = bitcast %struct.point474* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo475(%struct.point475* noalias sret(%struct.point475) align 4 %0, %struct.point475* noundef byval(%struct.point475) align 4 %1) #0 {
  %3 = bitcast %struct.point475* %0 to i8*
  %4 = bitcast %struct.point475* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo476(%struct.point476* noalias sret(%struct.point476) align 4 %0, %struct.point476* noundef byval(%struct.point476) align 4 %1) #0 {
  %3 = bitcast %struct.point476* %0 to i8*
  %4 = bitcast %struct.point476* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo477(%struct.point477* noalias sret(%struct.point477) align 4 %0, %struct.point477* noundef byval(%struct.point477) align 4 %1) #0 {
  %3 = bitcast %struct.point477* %0 to i8*
  %4 = bitcast %struct.point477* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo478(%struct.point478* noalias sret(%struct.point478) align 4 %0, %struct.point478* noundef byval(%struct.point478) align 4 %1) #0 {
  %3 = bitcast %struct.point478* %0 to i8*
  %4 = bitcast %struct.point478* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo479(%struct.point479* noalias sret(%struct.point479) align 4 %0, %struct.point479* noundef byval(%struct.point479) align 4 %1) #0 {
  %3 = bitcast %struct.point479* %0 to i8*
  %4 = bitcast %struct.point479* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo480(%struct.point480* noalias sret(%struct.point480) align 4 %0, %struct.point480* noundef byval(%struct.point480) align 4 %1) #0 {
  %3 = bitcast %struct.point480* %0 to i8*
  %4 = bitcast %struct.point480* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo481(%struct.point481* noalias sret(%struct.point481) align 4 %0, %struct.point481* noundef byval(%struct.point481) align 4 %1) #0 {
  %3 = bitcast %struct.point481* %0 to i8*
  %4 = bitcast %struct.point481* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo482(%struct.point482* noalias sret(%struct.point482) align 4 %0, %struct.point482* noundef byval(%struct.point482) align 4 %1) #0 {
  %3 = bitcast %struct.point482* %0 to i8*
  %4 = bitcast %struct.point482* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo483(%struct.point483* noalias sret(%struct.point483) align 4 %0, %struct.point483* noundef byval(%struct.point483) align 4 %1) #0 {
  %3 = bitcast %struct.point483* %0 to i8*
  %4 = bitcast %struct.point483* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo484(%struct.point484* noalias sret(%struct.point484) align 4 %0, %struct.point484* noundef byval(%struct.point484) align 4 %1) #0 {
  %3 = bitcast %struct.point484* %0 to i8*
  %4 = bitcast %struct.point484* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo485(%struct.point485* noalias sret(%struct.point485) align 4 %0, %struct.point485* noundef byval(%struct.point485) align 4 %1) #0 {
  %3 = bitcast %struct.point485* %0 to i8*
  %4 = bitcast %struct.point485* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo486(%struct.point486* noalias sret(%struct.point486) align 4 %0, %struct.point486* noundef byval(%struct.point486) align 4 %1) #0 {
  %3 = bitcast %struct.point486* %0 to i8*
  %4 = bitcast %struct.point486* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo487(%struct.point487* noalias sret(%struct.point487) align 4 %0, %struct.point487* noundef byval(%struct.point487) align 4 %1) #0 {
  %3 = bitcast %struct.point487* %0 to i8*
  %4 = bitcast %struct.point487* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo488(%struct.point488* noalias sret(%struct.point488) align 4 %0, %struct.point488* noundef byval(%struct.point488) align 4 %1) #0 {
  %3 = bitcast %struct.point488* %0 to i8*
  %4 = bitcast %struct.point488* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo489(%struct.point489* noalias sret(%struct.point489) align 4 %0, %struct.point489* noundef byval(%struct.point489) align 4 %1) #0 {
  %3 = bitcast %struct.point489* %0 to i8*
  %4 = bitcast %struct.point489* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo490(%struct.point490* noalias sret(%struct.point490) align 4 %0, %struct.point490* noundef byval(%struct.point490) align 4 %1) #0 {
  %3 = bitcast %struct.point490* %0 to i8*
  %4 = bitcast %struct.point490* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo491(%struct.point491* noalias sret(%struct.point491) align 4 %0, %struct.point491* noundef byval(%struct.point491) align 4 %1) #0 {
  %3 = bitcast %struct.point491* %0 to i8*
  %4 = bitcast %struct.point491* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo492(%struct.point492* noalias sret(%struct.point492) align 4 %0, %struct.point492* noundef byval(%struct.point492) align 4 %1) #0 {
  %3 = bitcast %struct.point492* %0 to i8*
  %4 = bitcast %struct.point492* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo493(%struct.point493* noalias sret(%struct.point493) align 4 %0, %struct.point493* noundef byval(%struct.point493) align 4 %1) #0 {
  %3 = bitcast %struct.point493* %0 to i8*
  %4 = bitcast %struct.point493* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo494(%struct.point494* noalias sret(%struct.point494) align 4 %0, %struct.point494* noundef byval(%struct.point494) align 4 %1) #0 {
  %3 = bitcast %struct.point494* %0 to i8*
  %4 = bitcast %struct.point494* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo495(%struct.point495* noalias sret(%struct.point495) align 4 %0, %struct.point495* noundef byval(%struct.point495) align 4 %1) #0 {
  %3 = bitcast %struct.point495* %0 to i8*
  %4 = bitcast %struct.point495* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo496(%struct.point496* noalias sret(%struct.point496) align 4 %0, %struct.point496* noundef byval(%struct.point496) align 4 %1) #0 {
  %3 = bitcast %struct.point496* %0 to i8*
  %4 = bitcast %struct.point496* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo497(%struct.point497* noalias sret(%struct.point497) align 4 %0, %struct.point497* noundef byval(%struct.point497) align 4 %1) #0 {
  %3 = bitcast %struct.point497* %0 to i8*
  %4 = bitcast %struct.point497* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo498(%struct.point498* noalias sret(%struct.point498) align 4 %0, %struct.point498* noundef byval(%struct.point498) align 4 %1) #0 {
  %3 = bitcast %struct.point498* %0 to i8*
  %4 = bitcast %struct.point498* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo499(%struct.point499* noalias sret(%struct.point499) align 4 %0, %struct.point499* noundef byval(%struct.point499) align 4 %1) #0 {
  %3 = bitcast %struct.point499* %0 to i8*
  %4 = bitcast %struct.point499* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo500(%struct.point500* noalias sret(%struct.point500) align 4 %0, %struct.point500* noundef byval(%struct.point500) align 4 %1) #0 {
  %3 = bitcast %struct.point500* %0 to i8*
  %4 = bitcast %struct.point500* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo501(%struct.point501* noalias sret(%struct.point501) align 4 %0, %struct.point501* noundef byval(%struct.point501) align 4 %1) #0 {
  %3 = bitcast %struct.point501* %0 to i8*
  %4 = bitcast %struct.point501* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo502(%struct.point502* noalias sret(%struct.point502) align 4 %0, %struct.point502* noundef byval(%struct.point502) align 4 %1) #0 {
  %3 = bitcast %struct.point502* %0 to i8*
  %4 = bitcast %struct.point502* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo503(%struct.point503* noalias sret(%struct.point503) align 4 %0, %struct.point503* noundef byval(%struct.point503) align 4 %1) #0 {
  %3 = bitcast %struct.point503* %0 to i8*
  %4 = bitcast %struct.point503* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo504(%struct.point504* noalias sret(%struct.point504) align 4 %0, %struct.point504* noundef byval(%struct.point504) align 4 %1) #0 {
  %3 = bitcast %struct.point504* %0 to i8*
  %4 = bitcast %struct.point504* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo505(%struct.point505* noalias sret(%struct.point505) align 4 %0, %struct.point505* noundef byval(%struct.point505) align 4 %1) #0 {
  %3 = bitcast %struct.point505* %0 to i8*
  %4 = bitcast %struct.point505* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo506(%struct.point506* noalias sret(%struct.point506) align 4 %0, %struct.point506* noundef byval(%struct.point506) align 4 %1) #0 {
  %3 = bitcast %struct.point506* %0 to i8*
  %4 = bitcast %struct.point506* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo507(%struct.point507* noalias sret(%struct.point507) align 4 %0, %struct.point507* noundef byval(%struct.point507) align 4 %1) #0 {
  %3 = bitcast %struct.point507* %0 to i8*
  %4 = bitcast %struct.point507* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo508(%struct.point508* noalias sret(%struct.point508) align 4 %0, %struct.point508* noundef byval(%struct.point508) align 4 %1) #0 {
  %3 = bitcast %struct.point508* %0 to i8*
  %4 = bitcast %struct.point508* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo509(%struct.point509* noalias sret(%struct.point509) align 4 %0, %struct.point509* noundef byval(%struct.point509) align 4 %1) #0 {
  %3 = bitcast %struct.point509* %0 to i8*
  %4 = bitcast %struct.point509* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo510(%struct.point510* noalias sret(%struct.point510) align 4 %0, %struct.point510* noundef byval(%struct.point510) align 4 %1) #0 {
  %3 = bitcast %struct.point510* %0 to i8*
  %4 = bitcast %struct.point510* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo511(%struct.point511* noalias sret(%struct.point511) align 4 %0, %struct.point511* noundef byval(%struct.point511) align 4 %1) #0 {
  %3 = bitcast %struct.point511* %0 to i8*
  %4 = bitcast %struct.point511* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo512(%struct.point512* noalias sret(%struct.point512) align 4 %0, %struct.point512* noundef byval(%struct.point512) align 4 %1) #0 {
  %3 = bitcast %struct.point512* %0 to i8*
  %4 = bitcast %struct.point512* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo513(%struct.point513* noalias sret(%struct.point513) align 4 %0, %struct.point513* noundef byval(%struct.point513) align 4 %1) #0 {
  %3 = bitcast %struct.point513* %0 to i8*
  %4 = bitcast %struct.point513* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo514(%struct.point514* noalias sret(%struct.point514) align 4 %0, %struct.point514* noundef byval(%struct.point514) align 4 %1) #0 {
  %3 = bitcast %struct.point514* %0 to i8*
  %4 = bitcast %struct.point514* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo515(%struct.point515* noalias sret(%struct.point515) align 4 %0, %struct.point515* noundef byval(%struct.point515) align 4 %1) #0 {
  %3 = bitcast %struct.point515* %0 to i8*
  %4 = bitcast %struct.point515* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo516(%struct.point516* noalias sret(%struct.point516) align 4 %0, %struct.point516* noundef byval(%struct.point516) align 4 %1) #0 {
  %3 = bitcast %struct.point516* %0 to i8*
  %4 = bitcast %struct.point516* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo517(%struct.point517* noalias sret(%struct.point517) align 4 %0, %struct.point517* noundef byval(%struct.point517) align 4 %1) #0 {
  %3 = bitcast %struct.point517* %0 to i8*
  %4 = bitcast %struct.point517* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo518(%struct.point518* noalias sret(%struct.point518) align 4 %0, %struct.point518* noundef byval(%struct.point518) align 4 %1) #0 {
  %3 = bitcast %struct.point518* %0 to i8*
  %4 = bitcast %struct.point518* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo519(%struct.point519* noalias sret(%struct.point519) align 4 %0, %struct.point519* noundef byval(%struct.point519) align 4 %1) #0 {
  %3 = bitcast %struct.point519* %0 to i8*
  %4 = bitcast %struct.point519* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo520(%struct.point520* noalias sret(%struct.point520) align 4 %0, %struct.point520* noundef byval(%struct.point520) align 4 %1) #0 {
  %3 = bitcast %struct.point520* %0 to i8*
  %4 = bitcast %struct.point520* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo521(%struct.point521* noalias sret(%struct.point521) align 4 %0, %struct.point521* noundef byval(%struct.point521) align 4 %1) #0 {
  %3 = bitcast %struct.point521* %0 to i8*
  %4 = bitcast %struct.point521* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo522(%struct.point522* noalias sret(%struct.point522) align 4 %0, %struct.point522* noundef byval(%struct.point522) align 4 %1) #0 {
  %3 = bitcast %struct.point522* %0 to i8*
  %4 = bitcast %struct.point522* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo523(%struct.point523* noalias sret(%struct.point523) align 4 %0, %struct.point523* noundef byval(%struct.point523) align 4 %1) #0 {
  %3 = bitcast %struct.point523* %0 to i8*
  %4 = bitcast %struct.point523* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo524(%struct.point524* noalias sret(%struct.point524) align 4 %0, %struct.point524* noundef byval(%struct.point524) align 4 %1) #0 {
  %3 = bitcast %struct.point524* %0 to i8*
  %4 = bitcast %struct.point524* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo525(%struct.point525* noalias sret(%struct.point525) align 4 %0, %struct.point525* noundef byval(%struct.point525) align 4 %1) #0 {
  %3 = bitcast %struct.point525* %0 to i8*
  %4 = bitcast %struct.point525* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo526(%struct.point526* noalias sret(%struct.point526) align 4 %0, %struct.point526* noundef byval(%struct.point526) align 4 %1) #0 {
  %3 = bitcast %struct.point526* %0 to i8*
  %4 = bitcast %struct.point526* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo527(%struct.point527* noalias sret(%struct.point527) align 4 %0, %struct.point527* noundef byval(%struct.point527) align 4 %1) #0 {
  %3 = bitcast %struct.point527* %0 to i8*
  %4 = bitcast %struct.point527* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo528(%struct.point528* noalias sret(%struct.point528) align 4 %0, %struct.point528* noundef byval(%struct.point528) align 4 %1) #0 {
  %3 = bitcast %struct.point528* %0 to i8*
  %4 = bitcast %struct.point528* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo529(%struct.point529* noalias sret(%struct.point529) align 4 %0, %struct.point529* noundef byval(%struct.point529) align 4 %1) #0 {
  %3 = bitcast %struct.point529* %0 to i8*
  %4 = bitcast %struct.point529* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo530(%struct.point530* noalias sret(%struct.point530) align 4 %0, %struct.point530* noundef byval(%struct.point530) align 4 %1) #0 {
  %3 = bitcast %struct.point530* %0 to i8*
  %4 = bitcast %struct.point530* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo531(%struct.point531* noalias sret(%struct.point531) align 4 %0, %struct.point531* noundef byval(%struct.point531) align 4 %1) #0 {
  %3 = bitcast %struct.point531* %0 to i8*
  %4 = bitcast %struct.point531* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo532(%struct.point532* noalias sret(%struct.point532) align 4 %0, %struct.point532* noundef byval(%struct.point532) align 4 %1) #0 {
  %3 = bitcast %struct.point532* %0 to i8*
  %4 = bitcast %struct.point532* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo533(%struct.point533* noalias sret(%struct.point533) align 4 %0, %struct.point533* noundef byval(%struct.point533) align 4 %1) #0 {
  %3 = bitcast %struct.point533* %0 to i8*
  %4 = bitcast %struct.point533* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo534(%struct.point534* noalias sret(%struct.point534) align 4 %0, %struct.point534* noundef byval(%struct.point534) align 4 %1) #0 {
  %3 = bitcast %struct.point534* %0 to i8*
  %4 = bitcast %struct.point534* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo535(%struct.point535* noalias sret(%struct.point535) align 4 %0, %struct.point535* noundef byval(%struct.point535) align 4 %1) #0 {
  %3 = bitcast %struct.point535* %0 to i8*
  %4 = bitcast %struct.point535* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo536(%struct.point536* noalias sret(%struct.point536) align 4 %0, %struct.point536* noundef byval(%struct.point536) align 4 %1) #0 {
  %3 = bitcast %struct.point536* %0 to i8*
  %4 = bitcast %struct.point536* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo537(%struct.point537* noalias sret(%struct.point537) align 4 %0, %struct.point537* noundef byval(%struct.point537) align 4 %1) #0 {
  %3 = bitcast %struct.point537* %0 to i8*
  %4 = bitcast %struct.point537* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo538(%struct.point538* noalias sret(%struct.point538) align 4 %0, %struct.point538* noundef byval(%struct.point538) align 4 %1) #0 {
  %3 = bitcast %struct.point538* %0 to i8*
  %4 = bitcast %struct.point538* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo539(%struct.point539* noalias sret(%struct.point539) align 4 %0, %struct.point539* noundef byval(%struct.point539) align 4 %1) #0 {
  %3 = bitcast %struct.point539* %0 to i8*
  %4 = bitcast %struct.point539* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo540(%struct.point540* noalias sret(%struct.point540) align 4 %0, %struct.point540* noundef byval(%struct.point540) align 4 %1) #0 {
  %3 = bitcast %struct.point540* %0 to i8*
  %4 = bitcast %struct.point540* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo541(%struct.point541* noalias sret(%struct.point541) align 4 %0, %struct.point541* noundef byval(%struct.point541) align 4 %1) #0 {
  %3 = bitcast %struct.point541* %0 to i8*
  %4 = bitcast %struct.point541* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo542(%struct.point542* noalias sret(%struct.point542) align 4 %0, %struct.point542* noundef byval(%struct.point542) align 4 %1) #0 {
  %3 = bitcast %struct.point542* %0 to i8*
  %4 = bitcast %struct.point542* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo543(%struct.point543* noalias sret(%struct.point543) align 4 %0, %struct.point543* noundef byval(%struct.point543) align 4 %1) #0 {
  %3 = bitcast %struct.point543* %0 to i8*
  %4 = bitcast %struct.point543* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo544(%struct.point544* noalias sret(%struct.point544) align 4 %0, %struct.point544* noundef byval(%struct.point544) align 4 %1) #0 {
  %3 = bitcast %struct.point544* %0 to i8*
  %4 = bitcast %struct.point544* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo545(%struct.point545* noalias sret(%struct.point545) align 4 %0, %struct.point545* noundef byval(%struct.point545) align 4 %1) #0 {
  %3 = bitcast %struct.point545* %0 to i8*
  %4 = bitcast %struct.point545* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo546(%struct.point546* noalias sret(%struct.point546) align 4 %0, %struct.point546* noundef byval(%struct.point546) align 4 %1) #0 {
  %3 = bitcast %struct.point546* %0 to i8*
  %4 = bitcast %struct.point546* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo547(%struct.point547* noalias sret(%struct.point547) align 4 %0, %struct.point547* noundef byval(%struct.point547) align 4 %1) #0 {
  %3 = bitcast %struct.point547* %0 to i8*
  %4 = bitcast %struct.point547* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo548(%struct.point548* noalias sret(%struct.point548) align 4 %0, %struct.point548* noundef byval(%struct.point548) align 4 %1) #0 {
  %3 = bitcast %struct.point548* %0 to i8*
  %4 = bitcast %struct.point548* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo549(%struct.point549* noalias sret(%struct.point549) align 4 %0, %struct.point549* noundef byval(%struct.point549) align 4 %1) #0 {
  %3 = bitcast %struct.point549* %0 to i8*
  %4 = bitcast %struct.point549* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo550(%struct.point550* noalias sret(%struct.point550) align 4 %0, %struct.point550* noundef byval(%struct.point550) align 4 %1) #0 {
  %3 = bitcast %struct.point550* %0 to i8*
  %4 = bitcast %struct.point550* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo551(%struct.point551* noalias sret(%struct.point551) align 4 %0, %struct.point551* noundef byval(%struct.point551) align 4 %1) #0 {
  %3 = bitcast %struct.point551* %0 to i8*
  %4 = bitcast %struct.point551* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo552(%struct.point552* noalias sret(%struct.point552) align 4 %0, %struct.point552* noundef byval(%struct.point552) align 4 %1) #0 {
  %3 = bitcast %struct.point552* %0 to i8*
  %4 = bitcast %struct.point552* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo553(%struct.point553* noalias sret(%struct.point553) align 4 %0, %struct.point553* noundef byval(%struct.point553) align 4 %1) #0 {
  %3 = bitcast %struct.point553* %0 to i8*
  %4 = bitcast %struct.point553* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo554(%struct.point554* noalias sret(%struct.point554) align 4 %0, %struct.point554* noundef byval(%struct.point554) align 4 %1) #0 {
  %3 = bitcast %struct.point554* %0 to i8*
  %4 = bitcast %struct.point554* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo555(%struct.point555* noalias sret(%struct.point555) align 4 %0, %struct.point555* noundef byval(%struct.point555) align 4 %1) #0 {
  %3 = bitcast %struct.point555* %0 to i8*
  %4 = bitcast %struct.point555* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo556(%struct.point556* noalias sret(%struct.point556) align 4 %0, %struct.point556* noundef byval(%struct.point556) align 4 %1) #0 {
  %3 = bitcast %struct.point556* %0 to i8*
  %4 = bitcast %struct.point556* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo557(%struct.point557* noalias sret(%struct.point557) align 4 %0, %struct.point557* noundef byval(%struct.point557) align 4 %1) #0 {
  %3 = bitcast %struct.point557* %0 to i8*
  %4 = bitcast %struct.point557* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo558(%struct.point558* noalias sret(%struct.point558) align 4 %0, %struct.point558* noundef byval(%struct.point558) align 4 %1) #0 {
  %3 = bitcast %struct.point558* %0 to i8*
  %4 = bitcast %struct.point558* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo559(%struct.point559* noalias sret(%struct.point559) align 4 %0, %struct.point559* noundef byval(%struct.point559) align 4 %1) #0 {
  %3 = bitcast %struct.point559* %0 to i8*
  %4 = bitcast %struct.point559* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo560(%struct.point560* noalias sret(%struct.point560) align 4 %0, %struct.point560* noundef byval(%struct.point560) align 4 %1) #0 {
  %3 = bitcast %struct.point560* %0 to i8*
  %4 = bitcast %struct.point560* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo561(%struct.point561* noalias sret(%struct.point561) align 4 %0, %struct.point561* noundef byval(%struct.point561) align 4 %1) #0 {
  %3 = bitcast %struct.point561* %0 to i8*
  %4 = bitcast %struct.point561* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo562(%struct.point562* noalias sret(%struct.point562) align 4 %0, %struct.point562* noundef byval(%struct.point562) align 4 %1) #0 {
  %3 = bitcast %struct.point562* %0 to i8*
  %4 = bitcast %struct.point562* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo563(%struct.point563* noalias sret(%struct.point563) align 4 %0, %struct.point563* noundef byval(%struct.point563) align 4 %1) #0 {
  %3 = bitcast %struct.point563* %0 to i8*
  %4 = bitcast %struct.point563* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo564(%struct.point564* noalias sret(%struct.point564) align 4 %0, %struct.point564* noundef byval(%struct.point564) align 4 %1) #0 {
  %3 = bitcast %struct.point564* %0 to i8*
  %4 = bitcast %struct.point564* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo565(%struct.point565* noalias sret(%struct.point565) align 4 %0, %struct.point565* noundef byval(%struct.point565) align 4 %1) #0 {
  %3 = bitcast %struct.point565* %0 to i8*
  %4 = bitcast %struct.point565* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo566(%struct.point566* noalias sret(%struct.point566) align 4 %0, %struct.point566* noundef byval(%struct.point566) align 4 %1) #0 {
  %3 = bitcast %struct.point566* %0 to i8*
  %4 = bitcast %struct.point566* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo567(%struct.point567* noalias sret(%struct.point567) align 4 %0, %struct.point567* noundef byval(%struct.point567) align 4 %1) #0 {
  %3 = bitcast %struct.point567* %0 to i8*
  %4 = bitcast %struct.point567* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo568(%struct.point568* noalias sret(%struct.point568) align 4 %0, %struct.point568* noundef byval(%struct.point568) align 4 %1) #0 {
  %3 = bitcast %struct.point568* %0 to i8*
  %4 = bitcast %struct.point568* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo569(%struct.point569* noalias sret(%struct.point569) align 4 %0, %struct.point569* noundef byval(%struct.point569) align 4 %1) #0 {
  %3 = bitcast %struct.point569* %0 to i8*
  %4 = bitcast %struct.point569* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo570(%struct.point570* noalias sret(%struct.point570) align 4 %0, %struct.point570* noundef byval(%struct.point570) align 4 %1) #0 {
  %3 = bitcast %struct.point570* %0 to i8*
  %4 = bitcast %struct.point570* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo571(%struct.point571* noalias sret(%struct.point571) align 4 %0, %struct.point571* noundef byval(%struct.point571) align 4 %1) #0 {
  %3 = bitcast %struct.point571* %0 to i8*
  %4 = bitcast %struct.point571* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo572(%struct.point572* noalias sret(%struct.point572) align 4 %0, %struct.point572* noundef byval(%struct.point572) align 4 %1) #0 {
  %3 = bitcast %struct.point572* %0 to i8*
  %4 = bitcast %struct.point572* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo573(%struct.point573* noalias sret(%struct.point573) align 4 %0, %struct.point573* noundef byval(%struct.point573) align 4 %1) #0 {
  %3 = bitcast %struct.point573* %0 to i8*
  %4 = bitcast %struct.point573* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo574(%struct.point574* noalias sret(%struct.point574) align 4 %0, %struct.point574* noundef byval(%struct.point574) align 4 %1) #0 {
  %3 = bitcast %struct.point574* %0 to i8*
  %4 = bitcast %struct.point574* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo575(%struct.point575* noalias sret(%struct.point575) align 4 %0, %struct.point575* noundef byval(%struct.point575) align 4 %1) #0 {
  %3 = bitcast %struct.point575* %0 to i8*
  %4 = bitcast %struct.point575* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo576(%struct.point576* noalias sret(%struct.point576) align 4 %0, %struct.point576* noundef byval(%struct.point576) align 4 %1) #0 {
  %3 = bitcast %struct.point576* %0 to i8*
  %4 = bitcast %struct.point576* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo577(%struct.point577* noalias sret(%struct.point577) align 4 %0, %struct.point577* noundef byval(%struct.point577) align 4 %1) #0 {
  %3 = bitcast %struct.point577* %0 to i8*
  %4 = bitcast %struct.point577* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo578(%struct.point578* noalias sret(%struct.point578) align 4 %0, %struct.point578* noundef byval(%struct.point578) align 4 %1) #0 {
  %3 = bitcast %struct.point578* %0 to i8*
  %4 = bitcast %struct.point578* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo579(%struct.point579* noalias sret(%struct.point579) align 4 %0, %struct.point579* noundef byval(%struct.point579) align 4 %1) #0 {
  %3 = bitcast %struct.point579* %0 to i8*
  %4 = bitcast %struct.point579* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo580(%struct.point580* noalias sret(%struct.point580) align 4 %0, %struct.point580* noundef byval(%struct.point580) align 4 %1) #0 {
  %3 = bitcast %struct.point580* %0 to i8*
  %4 = bitcast %struct.point580* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo581(%struct.point581* noalias sret(%struct.point581) align 4 %0, %struct.point581* noundef byval(%struct.point581) align 4 %1) #0 {
  %3 = bitcast %struct.point581* %0 to i8*
  %4 = bitcast %struct.point581* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo582(%struct.point582* noalias sret(%struct.point582) align 4 %0, %struct.point582* noundef byval(%struct.point582) align 4 %1) #0 {
  %3 = bitcast %struct.point582* %0 to i8*
  %4 = bitcast %struct.point582* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo583(%struct.point583* noalias sret(%struct.point583) align 4 %0, %struct.point583* noundef byval(%struct.point583) align 4 %1) #0 {
  %3 = bitcast %struct.point583* %0 to i8*
  %4 = bitcast %struct.point583* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo584(%struct.point584* noalias sret(%struct.point584) align 4 %0, %struct.point584* noundef byval(%struct.point584) align 4 %1) #0 {
  %3 = bitcast %struct.point584* %0 to i8*
  %4 = bitcast %struct.point584* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo585(%struct.point585* noalias sret(%struct.point585) align 4 %0, %struct.point585* noundef byval(%struct.point585) align 4 %1) #0 {
  %3 = bitcast %struct.point585* %0 to i8*
  %4 = bitcast %struct.point585* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo586(%struct.point586* noalias sret(%struct.point586) align 4 %0, %struct.point586* noundef byval(%struct.point586) align 4 %1) #0 {
  %3 = bitcast %struct.point586* %0 to i8*
  %4 = bitcast %struct.point586* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo587(%struct.point587* noalias sret(%struct.point587) align 4 %0, %struct.point587* noundef byval(%struct.point587) align 4 %1) #0 {
  %3 = bitcast %struct.point587* %0 to i8*
  %4 = bitcast %struct.point587* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo588(%struct.point588* noalias sret(%struct.point588) align 4 %0, %struct.point588* noundef byval(%struct.point588) align 4 %1) #0 {
  %3 = bitcast %struct.point588* %0 to i8*
  %4 = bitcast %struct.point588* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo589(%struct.point589* noalias sret(%struct.point589) align 4 %0, %struct.point589* noundef byval(%struct.point589) align 4 %1) #0 {
  %3 = bitcast %struct.point589* %0 to i8*
  %4 = bitcast %struct.point589* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo590(%struct.point590* noalias sret(%struct.point590) align 4 %0, %struct.point590* noundef byval(%struct.point590) align 4 %1) #0 {
  %3 = bitcast %struct.point590* %0 to i8*
  %4 = bitcast %struct.point590* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo591(%struct.point591* noalias sret(%struct.point591) align 4 %0, %struct.point591* noundef byval(%struct.point591) align 4 %1) #0 {
  %3 = bitcast %struct.point591* %0 to i8*
  %4 = bitcast %struct.point591* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo592(%struct.point592* noalias sret(%struct.point592) align 4 %0, %struct.point592* noundef byval(%struct.point592) align 4 %1) #0 {
  %3 = bitcast %struct.point592* %0 to i8*
  %4 = bitcast %struct.point592* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo593(%struct.point593* noalias sret(%struct.point593) align 4 %0, %struct.point593* noundef byval(%struct.point593) align 4 %1) #0 {
  %3 = bitcast %struct.point593* %0 to i8*
  %4 = bitcast %struct.point593* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo594(%struct.point594* noalias sret(%struct.point594) align 4 %0, %struct.point594* noundef byval(%struct.point594) align 4 %1) #0 {
  %3 = bitcast %struct.point594* %0 to i8*
  %4 = bitcast %struct.point594* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo595(%struct.point595* noalias sret(%struct.point595) align 4 %0, %struct.point595* noundef byval(%struct.point595) align 4 %1) #0 {
  %3 = bitcast %struct.point595* %0 to i8*
  %4 = bitcast %struct.point595* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo596(%struct.point596* noalias sret(%struct.point596) align 4 %0, %struct.point596* noundef byval(%struct.point596) align 4 %1) #0 {
  %3 = bitcast %struct.point596* %0 to i8*
  %4 = bitcast %struct.point596* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo597(%struct.point597* noalias sret(%struct.point597) align 4 %0, %struct.point597* noundef byval(%struct.point597) align 4 %1) #0 {
  %3 = bitcast %struct.point597* %0 to i8*
  %4 = bitcast %struct.point597* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo598(%struct.point598* noalias sret(%struct.point598) align 4 %0, %struct.point598* noundef byval(%struct.point598) align 4 %1) #0 {
  %3 = bitcast %struct.point598* %0 to i8*
  %4 = bitcast %struct.point598* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo599(%struct.point599* noalias sret(%struct.point599) align 4 %0, %struct.point599* noundef byval(%struct.point599) align 4 %1) #0 {
  %3 = bitcast %struct.point599* %0 to i8*
  %4 = bitcast %struct.point599* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo600(%struct.point600* noalias sret(%struct.point600) align 4 %0, %struct.point600* noundef byval(%struct.point600) align 4 %1) #0 {
  %3 = bitcast %struct.point600* %0 to i8*
  %4 = bitcast %struct.point600* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo601(%struct.point601* noalias sret(%struct.point601) align 4 %0, %struct.point601* noundef byval(%struct.point601) align 4 %1) #0 {
  %3 = bitcast %struct.point601* %0 to i8*
  %4 = bitcast %struct.point601* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo602(%struct.point602* noalias sret(%struct.point602) align 4 %0, %struct.point602* noundef byval(%struct.point602) align 4 %1) #0 {
  %3 = bitcast %struct.point602* %0 to i8*
  %4 = bitcast %struct.point602* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo603(%struct.point603* noalias sret(%struct.point603) align 4 %0, %struct.point603* noundef byval(%struct.point603) align 4 %1) #0 {
  %3 = bitcast %struct.point603* %0 to i8*
  %4 = bitcast %struct.point603* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo604(%struct.point604* noalias sret(%struct.point604) align 4 %0, %struct.point604* noundef byval(%struct.point604) align 4 %1) #0 {
  %3 = bitcast %struct.point604* %0 to i8*
  %4 = bitcast %struct.point604* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo605(%struct.point605* noalias sret(%struct.point605) align 4 %0, %struct.point605* noundef byval(%struct.point605) align 4 %1) #0 {
  %3 = bitcast %struct.point605* %0 to i8*
  %4 = bitcast %struct.point605* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo606(%struct.point606* noalias sret(%struct.point606) align 4 %0, %struct.point606* noundef byval(%struct.point606) align 4 %1) #0 {
  %3 = bitcast %struct.point606* %0 to i8*
  %4 = bitcast %struct.point606* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo607(%struct.point607* noalias sret(%struct.point607) align 4 %0, %struct.point607* noundef byval(%struct.point607) align 4 %1) #0 {
  %3 = bitcast %struct.point607* %0 to i8*
  %4 = bitcast %struct.point607* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo608(%struct.point608* noalias sret(%struct.point608) align 4 %0, %struct.point608* noundef byval(%struct.point608) align 4 %1) #0 {
  %3 = bitcast %struct.point608* %0 to i8*
  %4 = bitcast %struct.point608* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo609(%struct.point609* noalias sret(%struct.point609) align 4 %0, %struct.point609* noundef byval(%struct.point609) align 4 %1) #0 {
  %3 = bitcast %struct.point609* %0 to i8*
  %4 = bitcast %struct.point609* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo610(%struct.point610* noalias sret(%struct.point610) align 4 %0, %struct.point610* noundef byval(%struct.point610) align 4 %1) #0 {
  %3 = bitcast %struct.point610* %0 to i8*
  %4 = bitcast %struct.point610* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo611(%struct.point611* noalias sret(%struct.point611) align 4 %0, %struct.point611* noundef byval(%struct.point611) align 4 %1) #0 {
  %3 = bitcast %struct.point611* %0 to i8*
  %4 = bitcast %struct.point611* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo612(%struct.point612* noalias sret(%struct.point612) align 4 %0, %struct.point612* noundef byval(%struct.point612) align 4 %1) #0 {
  %3 = bitcast %struct.point612* %0 to i8*
  %4 = bitcast %struct.point612* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo613(%struct.point613* noalias sret(%struct.point613) align 4 %0, %struct.point613* noundef byval(%struct.point613) align 4 %1) #0 {
  %3 = bitcast %struct.point613* %0 to i8*
  %4 = bitcast %struct.point613* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo614(%struct.point614* noalias sret(%struct.point614) align 4 %0, %struct.point614* noundef byval(%struct.point614) align 4 %1) #0 {
  %3 = bitcast %struct.point614* %0 to i8*
  %4 = bitcast %struct.point614* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo615(%struct.point615* noalias sret(%struct.point615) align 4 %0, %struct.point615* noundef byval(%struct.point615) align 4 %1) #0 {
  %3 = bitcast %struct.point615* %0 to i8*
  %4 = bitcast %struct.point615* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo616(%struct.point616* noalias sret(%struct.point616) align 4 %0, %struct.point616* noundef byval(%struct.point616) align 4 %1) #0 {
  %3 = bitcast %struct.point616* %0 to i8*
  %4 = bitcast %struct.point616* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo617(%struct.point617* noalias sret(%struct.point617) align 4 %0, %struct.point617* noundef byval(%struct.point617) align 4 %1) #0 {
  %3 = bitcast %struct.point617* %0 to i8*
  %4 = bitcast %struct.point617* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo618(%struct.point618* noalias sret(%struct.point618) align 4 %0, %struct.point618* noundef byval(%struct.point618) align 4 %1) #0 {
  %3 = bitcast %struct.point618* %0 to i8*
  %4 = bitcast %struct.point618* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo619(%struct.point619* noalias sret(%struct.point619) align 4 %0, %struct.point619* noundef byval(%struct.point619) align 4 %1) #0 {
  %3 = bitcast %struct.point619* %0 to i8*
  %4 = bitcast %struct.point619* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo620(%struct.point620* noalias sret(%struct.point620) align 4 %0, %struct.point620* noundef byval(%struct.point620) align 4 %1) #0 {
  %3 = bitcast %struct.point620* %0 to i8*
  %4 = bitcast %struct.point620* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo621(%struct.point621* noalias sret(%struct.point621) align 4 %0, %struct.point621* noundef byval(%struct.point621) align 4 %1) #0 {
  %3 = bitcast %struct.point621* %0 to i8*
  %4 = bitcast %struct.point621* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo622(%struct.point622* noalias sret(%struct.point622) align 4 %0, %struct.point622* noundef byval(%struct.point622) align 4 %1) #0 {
  %3 = bitcast %struct.point622* %0 to i8*
  %4 = bitcast %struct.point622* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo623(%struct.point623* noalias sret(%struct.point623) align 4 %0, %struct.point623* noundef byval(%struct.point623) align 4 %1) #0 {
  %3 = bitcast %struct.point623* %0 to i8*
  %4 = bitcast %struct.point623* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo624(%struct.point624* noalias sret(%struct.point624) align 4 %0, %struct.point624* noundef byval(%struct.point624) align 4 %1) #0 {
  %3 = bitcast %struct.point624* %0 to i8*
  %4 = bitcast %struct.point624* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo625(%struct.point625* noalias sret(%struct.point625) align 4 %0, %struct.point625* noundef byval(%struct.point625) align 4 %1) #0 {
  %3 = bitcast %struct.point625* %0 to i8*
  %4 = bitcast %struct.point625* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo626(%struct.point626* noalias sret(%struct.point626) align 4 %0, %struct.point626* noundef byval(%struct.point626) align 4 %1) #0 {
  %3 = bitcast %struct.point626* %0 to i8*
  %4 = bitcast %struct.point626* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo627(%struct.point627* noalias sret(%struct.point627) align 4 %0, %struct.point627* noundef byval(%struct.point627) align 4 %1) #0 {
  %3 = bitcast %struct.point627* %0 to i8*
  %4 = bitcast %struct.point627* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo628(%struct.point628* noalias sret(%struct.point628) align 4 %0, %struct.point628* noundef byval(%struct.point628) align 4 %1) #0 {
  %3 = bitcast %struct.point628* %0 to i8*
  %4 = bitcast %struct.point628* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo629(%struct.point629* noalias sret(%struct.point629) align 4 %0, %struct.point629* noundef byval(%struct.point629) align 4 %1) #0 {
  %3 = bitcast %struct.point629* %0 to i8*
  %4 = bitcast %struct.point629* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo630(%struct.point630* noalias sret(%struct.point630) align 4 %0, %struct.point630* noundef byval(%struct.point630) align 4 %1) #0 {
  %3 = bitcast %struct.point630* %0 to i8*
  %4 = bitcast %struct.point630* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo631(%struct.point631* noalias sret(%struct.point631) align 4 %0, %struct.point631* noundef byval(%struct.point631) align 4 %1) #0 {
  %3 = bitcast %struct.point631* %0 to i8*
  %4 = bitcast %struct.point631* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo632(%struct.point632* noalias sret(%struct.point632) align 4 %0, %struct.point632* noundef byval(%struct.point632) align 4 %1) #0 {
  %3 = bitcast %struct.point632* %0 to i8*
  %4 = bitcast %struct.point632* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo633(%struct.point633* noalias sret(%struct.point633) align 4 %0, %struct.point633* noundef byval(%struct.point633) align 4 %1) #0 {
  %3 = bitcast %struct.point633* %0 to i8*
  %4 = bitcast %struct.point633* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo634(%struct.point634* noalias sret(%struct.point634) align 4 %0, %struct.point634* noundef byval(%struct.point634) align 4 %1) #0 {
  %3 = bitcast %struct.point634* %0 to i8*
  %4 = bitcast %struct.point634* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo635(%struct.point635* noalias sret(%struct.point635) align 4 %0, %struct.point635* noundef byval(%struct.point635) align 4 %1) #0 {
  %3 = bitcast %struct.point635* %0 to i8*
  %4 = bitcast %struct.point635* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo636(%struct.point636* noalias sret(%struct.point636) align 4 %0, %struct.point636* noundef byval(%struct.point636) align 4 %1) #0 {
  %3 = bitcast %struct.point636* %0 to i8*
  %4 = bitcast %struct.point636* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo637(%struct.point637* noalias sret(%struct.point637) align 4 %0, %struct.point637* noundef byval(%struct.point637) align 4 %1) #0 {
  %3 = bitcast %struct.point637* %0 to i8*
  %4 = bitcast %struct.point637* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo638(%struct.point638* noalias sret(%struct.point638) align 4 %0, %struct.point638* noundef byval(%struct.point638) align 4 %1) #0 {
  %3 = bitcast %struct.point638* %0 to i8*
  %4 = bitcast %struct.point638* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo639(%struct.point639* noalias sret(%struct.point639) align 4 %0, %struct.point639* noundef byval(%struct.point639) align 4 %1) #0 {
  %3 = bitcast %struct.point639* %0 to i8*
  %4 = bitcast %struct.point639* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo640(%struct.point640* noalias sret(%struct.point640) align 4 %0, %struct.point640* noundef byval(%struct.point640) align 4 %1) #0 {
  %3 = bitcast %struct.point640* %0 to i8*
  %4 = bitcast %struct.point640* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo641(%struct.point641* noalias sret(%struct.point641) align 4 %0, %struct.point641* noundef byval(%struct.point641) align 4 %1) #0 {
  %3 = bitcast %struct.point641* %0 to i8*
  %4 = bitcast %struct.point641* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo642(%struct.point642* noalias sret(%struct.point642) align 4 %0, %struct.point642* noundef byval(%struct.point642) align 4 %1) #0 {
  %3 = bitcast %struct.point642* %0 to i8*
  %4 = bitcast %struct.point642* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo643(%struct.point643* noalias sret(%struct.point643) align 4 %0, %struct.point643* noundef byval(%struct.point643) align 4 %1) #0 {
  %3 = bitcast %struct.point643* %0 to i8*
  %4 = bitcast %struct.point643* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo644(%struct.point644* noalias sret(%struct.point644) align 4 %0, %struct.point644* noundef byval(%struct.point644) align 4 %1) #0 {
  %3 = bitcast %struct.point644* %0 to i8*
  %4 = bitcast %struct.point644* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo645(%struct.point645* noalias sret(%struct.point645) align 4 %0, %struct.point645* noundef byval(%struct.point645) align 4 %1) #0 {
  %3 = bitcast %struct.point645* %0 to i8*
  %4 = bitcast %struct.point645* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo646(%struct.point646* noalias sret(%struct.point646) align 4 %0, %struct.point646* noundef byval(%struct.point646) align 4 %1) #0 {
  %3 = bitcast %struct.point646* %0 to i8*
  %4 = bitcast %struct.point646* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo647(%struct.point647* noalias sret(%struct.point647) align 4 %0, %struct.point647* noundef byval(%struct.point647) align 4 %1) #0 {
  %3 = bitcast %struct.point647* %0 to i8*
  %4 = bitcast %struct.point647* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo648(%struct.point648* noalias sret(%struct.point648) align 4 %0, %struct.point648* noundef byval(%struct.point648) align 4 %1) #0 {
  %3 = bitcast %struct.point648* %0 to i8*
  %4 = bitcast %struct.point648* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo649(%struct.point649* noalias sret(%struct.point649) align 4 %0, %struct.point649* noundef byval(%struct.point649) align 4 %1) #0 {
  %3 = bitcast %struct.point649* %0 to i8*
  %4 = bitcast %struct.point649* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo650(%struct.point650* noalias sret(%struct.point650) align 4 %0, %struct.point650* noundef byval(%struct.point650) align 4 %1) #0 {
  %3 = bitcast %struct.point650* %0 to i8*
  %4 = bitcast %struct.point650* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo651(%struct.point651* noalias sret(%struct.point651) align 4 %0, %struct.point651* noundef byval(%struct.point651) align 4 %1) #0 {
  %3 = bitcast %struct.point651* %0 to i8*
  %4 = bitcast %struct.point651* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo652(%struct.point652* noalias sret(%struct.point652) align 4 %0, %struct.point652* noundef byval(%struct.point652) align 4 %1) #0 {
  %3 = bitcast %struct.point652* %0 to i8*
  %4 = bitcast %struct.point652* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo653(%struct.point653* noalias sret(%struct.point653) align 4 %0, %struct.point653* noundef byval(%struct.point653) align 4 %1) #0 {
  %3 = bitcast %struct.point653* %0 to i8*
  %4 = bitcast %struct.point653* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo654(%struct.point654* noalias sret(%struct.point654) align 4 %0, %struct.point654* noundef byval(%struct.point654) align 4 %1) #0 {
  %3 = bitcast %struct.point654* %0 to i8*
  %4 = bitcast %struct.point654* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo655(%struct.point655* noalias sret(%struct.point655) align 4 %0, %struct.point655* noundef byval(%struct.point655) align 4 %1) #0 {
  %3 = bitcast %struct.point655* %0 to i8*
  %4 = bitcast %struct.point655* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo656(%struct.point656* noalias sret(%struct.point656) align 4 %0, %struct.point656* noundef byval(%struct.point656) align 4 %1) #0 {
  %3 = bitcast %struct.point656* %0 to i8*
  %4 = bitcast %struct.point656* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo657(%struct.point657* noalias sret(%struct.point657) align 4 %0, %struct.point657* noundef byval(%struct.point657) align 4 %1) #0 {
  %3 = bitcast %struct.point657* %0 to i8*
  %4 = bitcast %struct.point657* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo658(%struct.point658* noalias sret(%struct.point658) align 4 %0, %struct.point658* noundef byval(%struct.point658) align 4 %1) #0 {
  %3 = bitcast %struct.point658* %0 to i8*
  %4 = bitcast %struct.point658* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo659(%struct.point659* noalias sret(%struct.point659) align 4 %0, %struct.point659* noundef byval(%struct.point659) align 4 %1) #0 {
  %3 = bitcast %struct.point659* %0 to i8*
  %4 = bitcast %struct.point659* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo660(%struct.point660* noalias sret(%struct.point660) align 4 %0, %struct.point660* noundef byval(%struct.point660) align 4 %1) #0 {
  %3 = bitcast %struct.point660* %0 to i8*
  %4 = bitcast %struct.point660* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo661(%struct.point661* noalias sret(%struct.point661) align 4 %0, %struct.point661* noundef byval(%struct.point661) align 4 %1) #0 {
  %3 = bitcast %struct.point661* %0 to i8*
  %4 = bitcast %struct.point661* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo662(%struct.point662* noalias sret(%struct.point662) align 4 %0, %struct.point662* noundef byval(%struct.point662) align 4 %1) #0 {
  %3 = bitcast %struct.point662* %0 to i8*
  %4 = bitcast %struct.point662* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo663(%struct.point663* noalias sret(%struct.point663) align 4 %0, %struct.point663* noundef byval(%struct.point663) align 4 %1) #0 {
  %3 = bitcast %struct.point663* %0 to i8*
  %4 = bitcast %struct.point663* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo664(%struct.point664* noalias sret(%struct.point664) align 4 %0, %struct.point664* noundef byval(%struct.point664) align 4 %1) #0 {
  %3 = bitcast %struct.point664* %0 to i8*
  %4 = bitcast %struct.point664* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo665(%struct.point665* noalias sret(%struct.point665) align 4 %0, %struct.point665* noundef byval(%struct.point665) align 4 %1) #0 {
  %3 = bitcast %struct.point665* %0 to i8*
  %4 = bitcast %struct.point665* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo666(%struct.point666* noalias sret(%struct.point666) align 4 %0, %struct.point666* noundef byval(%struct.point666) align 4 %1) #0 {
  %3 = bitcast %struct.point666* %0 to i8*
  %4 = bitcast %struct.point666* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo667(%struct.point667* noalias sret(%struct.point667) align 4 %0, %struct.point667* noundef byval(%struct.point667) align 4 %1) #0 {
  %3 = bitcast %struct.point667* %0 to i8*
  %4 = bitcast %struct.point667* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo668(%struct.point668* noalias sret(%struct.point668) align 4 %0, %struct.point668* noundef byval(%struct.point668) align 4 %1) #0 {
  %3 = bitcast %struct.point668* %0 to i8*
  %4 = bitcast %struct.point668* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo669(%struct.point669* noalias sret(%struct.point669) align 4 %0, %struct.point669* noundef byval(%struct.point669) align 4 %1) #0 {
  %3 = bitcast %struct.point669* %0 to i8*
  %4 = bitcast %struct.point669* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo670(%struct.point670* noalias sret(%struct.point670) align 4 %0, %struct.point670* noundef byval(%struct.point670) align 4 %1) #0 {
  %3 = bitcast %struct.point670* %0 to i8*
  %4 = bitcast %struct.point670* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo671(%struct.point671* noalias sret(%struct.point671) align 4 %0, %struct.point671* noundef byval(%struct.point671) align 4 %1) #0 {
  %3 = bitcast %struct.point671* %0 to i8*
  %4 = bitcast %struct.point671* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo672(%struct.point672* noalias sret(%struct.point672) align 4 %0, %struct.point672* noundef byval(%struct.point672) align 4 %1) #0 {
  %3 = bitcast %struct.point672* %0 to i8*
  %4 = bitcast %struct.point672* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo673(%struct.point673* noalias sret(%struct.point673) align 4 %0, %struct.point673* noundef byval(%struct.point673) align 4 %1) #0 {
  %3 = bitcast %struct.point673* %0 to i8*
  %4 = bitcast %struct.point673* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo674(%struct.point674* noalias sret(%struct.point674) align 4 %0, %struct.point674* noundef byval(%struct.point674) align 4 %1) #0 {
  %3 = bitcast %struct.point674* %0 to i8*
  %4 = bitcast %struct.point674* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo675(%struct.point675* noalias sret(%struct.point675) align 4 %0, %struct.point675* noundef byval(%struct.point675) align 4 %1) #0 {
  %3 = bitcast %struct.point675* %0 to i8*
  %4 = bitcast %struct.point675* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo676(%struct.point676* noalias sret(%struct.point676) align 4 %0, %struct.point676* noundef byval(%struct.point676) align 4 %1) #0 {
  %3 = bitcast %struct.point676* %0 to i8*
  %4 = bitcast %struct.point676* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo677(%struct.point677* noalias sret(%struct.point677) align 4 %0, %struct.point677* noundef byval(%struct.point677) align 4 %1) #0 {
  %3 = bitcast %struct.point677* %0 to i8*
  %4 = bitcast %struct.point677* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo678(%struct.point678* noalias sret(%struct.point678) align 4 %0, %struct.point678* noundef byval(%struct.point678) align 4 %1) #0 {
  %3 = bitcast %struct.point678* %0 to i8*
  %4 = bitcast %struct.point678* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo679(%struct.point679* noalias sret(%struct.point679) align 4 %0, %struct.point679* noundef byval(%struct.point679) align 4 %1) #0 {
  %3 = bitcast %struct.point679* %0 to i8*
  %4 = bitcast %struct.point679* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo680(%struct.point680* noalias sret(%struct.point680) align 4 %0, %struct.point680* noundef byval(%struct.point680) align 4 %1) #0 {
  %3 = bitcast %struct.point680* %0 to i8*
  %4 = bitcast %struct.point680* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo681(%struct.point681* noalias sret(%struct.point681) align 4 %0, %struct.point681* noundef byval(%struct.point681) align 4 %1) #0 {
  %3 = bitcast %struct.point681* %0 to i8*
  %4 = bitcast %struct.point681* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo682(%struct.point682* noalias sret(%struct.point682) align 4 %0, %struct.point682* noundef byval(%struct.point682) align 4 %1) #0 {
  %3 = bitcast %struct.point682* %0 to i8*
  %4 = bitcast %struct.point682* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo683(%struct.point683* noalias sret(%struct.point683) align 4 %0, %struct.point683* noundef byval(%struct.point683) align 4 %1) #0 {
  %3 = bitcast %struct.point683* %0 to i8*
  %4 = bitcast %struct.point683* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo684(%struct.point684* noalias sret(%struct.point684) align 4 %0, %struct.point684* noundef byval(%struct.point684) align 4 %1) #0 {
  %3 = bitcast %struct.point684* %0 to i8*
  %4 = bitcast %struct.point684* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo685(%struct.point685* noalias sret(%struct.point685) align 4 %0, %struct.point685* noundef byval(%struct.point685) align 4 %1) #0 {
  %3 = bitcast %struct.point685* %0 to i8*
  %4 = bitcast %struct.point685* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo686(%struct.point686* noalias sret(%struct.point686) align 4 %0, %struct.point686* noundef byval(%struct.point686) align 4 %1) #0 {
  %3 = bitcast %struct.point686* %0 to i8*
  %4 = bitcast %struct.point686* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo687(%struct.point687* noalias sret(%struct.point687) align 4 %0, %struct.point687* noundef byval(%struct.point687) align 4 %1) #0 {
  %3 = bitcast %struct.point687* %0 to i8*
  %4 = bitcast %struct.point687* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo688(%struct.point688* noalias sret(%struct.point688) align 4 %0, %struct.point688* noundef byval(%struct.point688) align 4 %1) #0 {
  %3 = bitcast %struct.point688* %0 to i8*
  %4 = bitcast %struct.point688* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo689(%struct.point689* noalias sret(%struct.point689) align 4 %0, %struct.point689* noundef byval(%struct.point689) align 4 %1) #0 {
  %3 = bitcast %struct.point689* %0 to i8*
  %4 = bitcast %struct.point689* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo690(%struct.point690* noalias sret(%struct.point690) align 4 %0, %struct.point690* noundef byval(%struct.point690) align 4 %1) #0 {
  %3 = bitcast %struct.point690* %0 to i8*
  %4 = bitcast %struct.point690* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo691(%struct.point691* noalias sret(%struct.point691) align 4 %0, %struct.point691* noundef byval(%struct.point691) align 4 %1) #0 {
  %3 = bitcast %struct.point691* %0 to i8*
  %4 = bitcast %struct.point691* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo692(%struct.point692* noalias sret(%struct.point692) align 4 %0, %struct.point692* noundef byval(%struct.point692) align 4 %1) #0 {
  %3 = bitcast %struct.point692* %0 to i8*
  %4 = bitcast %struct.point692* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo693(%struct.point693* noalias sret(%struct.point693) align 4 %0, %struct.point693* noundef byval(%struct.point693) align 4 %1) #0 {
  %3 = bitcast %struct.point693* %0 to i8*
  %4 = bitcast %struct.point693* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo694(%struct.point694* noalias sret(%struct.point694) align 4 %0, %struct.point694* noundef byval(%struct.point694) align 4 %1) #0 {
  %3 = bitcast %struct.point694* %0 to i8*
  %4 = bitcast %struct.point694* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo695(%struct.point695* noalias sret(%struct.point695) align 4 %0, %struct.point695* noundef byval(%struct.point695) align 4 %1) #0 {
  %3 = bitcast %struct.point695* %0 to i8*
  %4 = bitcast %struct.point695* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo696(%struct.point696* noalias sret(%struct.point696) align 4 %0, %struct.point696* noundef byval(%struct.point696) align 4 %1) #0 {
  %3 = bitcast %struct.point696* %0 to i8*
  %4 = bitcast %struct.point696* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo697(%struct.point697* noalias sret(%struct.point697) align 4 %0, %struct.point697* noundef byval(%struct.point697) align 4 %1) #0 {
  %3 = bitcast %struct.point697* %0 to i8*
  %4 = bitcast %struct.point697* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo698(%struct.point698* noalias sret(%struct.point698) align 4 %0, %struct.point698* noundef byval(%struct.point698) align 4 %1) #0 {
  %3 = bitcast %struct.point698* %0 to i8*
  %4 = bitcast %struct.point698* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo699(%struct.point699* noalias sret(%struct.point699) align 4 %0, %struct.point699* noundef byval(%struct.point699) align 4 %1) #0 {
  %3 = bitcast %struct.point699* %0 to i8*
  %4 = bitcast %struct.point699* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo700(%struct.point700* noalias sret(%struct.point700) align 4 %0, %struct.point700* noundef byval(%struct.point700) align 4 %1) #0 {
  %3 = bitcast %struct.point700* %0 to i8*
  %4 = bitcast %struct.point700* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo701(%struct.point701* noalias sret(%struct.point701) align 4 %0, %struct.point701* noundef byval(%struct.point701) align 4 %1) #0 {
  %3 = bitcast %struct.point701* %0 to i8*
  %4 = bitcast %struct.point701* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo702(%struct.point702* noalias sret(%struct.point702) align 4 %0, %struct.point702* noundef byval(%struct.point702) align 4 %1) #0 {
  %3 = bitcast %struct.point702* %0 to i8*
  %4 = bitcast %struct.point702* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo703(%struct.point703* noalias sret(%struct.point703) align 4 %0, %struct.point703* noundef byval(%struct.point703) align 4 %1) #0 {
  %3 = bitcast %struct.point703* %0 to i8*
  %4 = bitcast %struct.point703* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo704(%struct.point704* noalias sret(%struct.point704) align 4 %0, %struct.point704* noundef byval(%struct.point704) align 4 %1) #0 {
  %3 = bitcast %struct.point704* %0 to i8*
  %4 = bitcast %struct.point704* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo705(%struct.point705* noalias sret(%struct.point705) align 4 %0, %struct.point705* noundef byval(%struct.point705) align 4 %1) #0 {
  %3 = bitcast %struct.point705* %0 to i8*
  %4 = bitcast %struct.point705* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo706(%struct.point706* noalias sret(%struct.point706) align 4 %0, %struct.point706* noundef byval(%struct.point706) align 4 %1) #0 {
  %3 = bitcast %struct.point706* %0 to i8*
  %4 = bitcast %struct.point706* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo707(%struct.point707* noalias sret(%struct.point707) align 4 %0, %struct.point707* noundef byval(%struct.point707) align 4 %1) #0 {
  %3 = bitcast %struct.point707* %0 to i8*
  %4 = bitcast %struct.point707* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo708(%struct.point708* noalias sret(%struct.point708) align 4 %0, %struct.point708* noundef byval(%struct.point708) align 4 %1) #0 {
  %3 = bitcast %struct.point708* %0 to i8*
  %4 = bitcast %struct.point708* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo709(%struct.point709* noalias sret(%struct.point709) align 4 %0, %struct.point709* noundef byval(%struct.point709) align 4 %1) #0 {
  %3 = bitcast %struct.point709* %0 to i8*
  %4 = bitcast %struct.point709* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo710(%struct.point710* noalias sret(%struct.point710) align 4 %0, %struct.point710* noundef byval(%struct.point710) align 4 %1) #0 {
  %3 = bitcast %struct.point710* %0 to i8*
  %4 = bitcast %struct.point710* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo711(%struct.point711* noalias sret(%struct.point711) align 4 %0, %struct.point711* noundef byval(%struct.point711) align 4 %1) #0 {
  %3 = bitcast %struct.point711* %0 to i8*
  %4 = bitcast %struct.point711* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo712(%struct.point712* noalias sret(%struct.point712) align 4 %0, %struct.point712* noundef byval(%struct.point712) align 4 %1) #0 {
  %3 = bitcast %struct.point712* %0 to i8*
  %4 = bitcast %struct.point712* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo713(%struct.point713* noalias sret(%struct.point713) align 4 %0, %struct.point713* noundef byval(%struct.point713) align 4 %1) #0 {
  %3 = bitcast %struct.point713* %0 to i8*
  %4 = bitcast %struct.point713* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo714(%struct.point714* noalias sret(%struct.point714) align 4 %0, %struct.point714* noundef byval(%struct.point714) align 4 %1) #0 {
  %3 = bitcast %struct.point714* %0 to i8*
  %4 = bitcast %struct.point714* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo715(%struct.point715* noalias sret(%struct.point715) align 4 %0, %struct.point715* noundef byval(%struct.point715) align 4 %1) #0 {
  %3 = bitcast %struct.point715* %0 to i8*
  %4 = bitcast %struct.point715* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo716(%struct.point716* noalias sret(%struct.point716) align 4 %0, %struct.point716* noundef byval(%struct.point716) align 4 %1) #0 {
  %3 = bitcast %struct.point716* %0 to i8*
  %4 = bitcast %struct.point716* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo717(%struct.point717* noalias sret(%struct.point717) align 4 %0, %struct.point717* noundef byval(%struct.point717) align 4 %1) #0 {
  %3 = bitcast %struct.point717* %0 to i8*
  %4 = bitcast %struct.point717* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo718(%struct.point718* noalias sret(%struct.point718) align 4 %0, %struct.point718* noundef byval(%struct.point718) align 4 %1) #0 {
  %3 = bitcast %struct.point718* %0 to i8*
  %4 = bitcast %struct.point718* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo719(%struct.point719* noalias sret(%struct.point719) align 4 %0, %struct.point719* noundef byval(%struct.point719) align 4 %1) #0 {
  %3 = bitcast %struct.point719* %0 to i8*
  %4 = bitcast %struct.point719* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo720(%struct.point720* noalias sret(%struct.point720) align 4 %0, %struct.point720* noundef byval(%struct.point720) align 4 %1) #0 {
  %3 = bitcast %struct.point720* %0 to i8*
  %4 = bitcast %struct.point720* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo721(%struct.point721* noalias sret(%struct.point721) align 4 %0, %struct.point721* noundef byval(%struct.point721) align 4 %1) #0 {
  %3 = bitcast %struct.point721* %0 to i8*
  %4 = bitcast %struct.point721* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo722(%struct.point722* noalias sret(%struct.point722) align 4 %0, %struct.point722* noundef byval(%struct.point722) align 4 %1) #0 {
  %3 = bitcast %struct.point722* %0 to i8*
  %4 = bitcast %struct.point722* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo723(%struct.point723* noalias sret(%struct.point723) align 4 %0, %struct.point723* noundef byval(%struct.point723) align 4 %1) #0 {
  %3 = bitcast %struct.point723* %0 to i8*
  %4 = bitcast %struct.point723* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo724(%struct.point724* noalias sret(%struct.point724) align 4 %0, %struct.point724* noundef byval(%struct.point724) align 4 %1) #0 {
  %3 = bitcast %struct.point724* %0 to i8*
  %4 = bitcast %struct.point724* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo725(%struct.point725* noalias sret(%struct.point725) align 4 %0, %struct.point725* noundef byval(%struct.point725) align 4 %1) #0 {
  %3 = bitcast %struct.point725* %0 to i8*
  %4 = bitcast %struct.point725* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo726(%struct.point726* noalias sret(%struct.point726) align 4 %0, %struct.point726* noundef byval(%struct.point726) align 4 %1) #0 {
  %3 = bitcast %struct.point726* %0 to i8*
  %4 = bitcast %struct.point726* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo727(%struct.point727* noalias sret(%struct.point727) align 4 %0, %struct.point727* noundef byval(%struct.point727) align 4 %1) #0 {
  %3 = bitcast %struct.point727* %0 to i8*
  %4 = bitcast %struct.point727* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo728(%struct.point728* noalias sret(%struct.point728) align 4 %0, %struct.point728* noundef byval(%struct.point728) align 4 %1) #0 {
  %3 = bitcast %struct.point728* %0 to i8*
  %4 = bitcast %struct.point728* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo729(%struct.point729* noalias sret(%struct.point729) align 4 %0, %struct.point729* noundef byval(%struct.point729) align 4 %1) #0 {
  %3 = bitcast %struct.point729* %0 to i8*
  %4 = bitcast %struct.point729* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo730(%struct.point730* noalias sret(%struct.point730) align 4 %0, %struct.point730* noundef byval(%struct.point730) align 4 %1) #0 {
  %3 = bitcast %struct.point730* %0 to i8*
  %4 = bitcast %struct.point730* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo731(%struct.point731* noalias sret(%struct.point731) align 4 %0, %struct.point731* noundef byval(%struct.point731) align 4 %1) #0 {
  %3 = bitcast %struct.point731* %0 to i8*
  %4 = bitcast %struct.point731* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo732(%struct.point732* noalias sret(%struct.point732) align 4 %0, %struct.point732* noundef byval(%struct.point732) align 4 %1) #0 {
  %3 = bitcast %struct.point732* %0 to i8*
  %4 = bitcast %struct.point732* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo733(%struct.point733* noalias sret(%struct.point733) align 4 %0, %struct.point733* noundef byval(%struct.point733) align 4 %1) #0 {
  %3 = bitcast %struct.point733* %0 to i8*
  %4 = bitcast %struct.point733* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo734(%struct.point734* noalias sret(%struct.point734) align 4 %0, %struct.point734* noundef byval(%struct.point734) align 4 %1) #0 {
  %3 = bitcast %struct.point734* %0 to i8*
  %4 = bitcast %struct.point734* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo735(%struct.point735* noalias sret(%struct.point735) align 4 %0, %struct.point735* noundef byval(%struct.point735) align 4 %1) #0 {
  %3 = bitcast %struct.point735* %0 to i8*
  %4 = bitcast %struct.point735* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo736(%struct.point736* noalias sret(%struct.point736) align 4 %0, %struct.point736* noundef byval(%struct.point736) align 4 %1) #0 {
  %3 = bitcast %struct.point736* %0 to i8*
  %4 = bitcast %struct.point736* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo737(%struct.point737* noalias sret(%struct.point737) align 4 %0, %struct.point737* noundef byval(%struct.point737) align 4 %1) #0 {
  %3 = bitcast %struct.point737* %0 to i8*
  %4 = bitcast %struct.point737* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo738(%struct.point738* noalias sret(%struct.point738) align 4 %0, %struct.point738* noundef byval(%struct.point738) align 4 %1) #0 {
  %3 = bitcast %struct.point738* %0 to i8*
  %4 = bitcast %struct.point738* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo739(%struct.point739* noalias sret(%struct.point739) align 4 %0, %struct.point739* noundef byval(%struct.point739) align 4 %1) #0 {
  %3 = bitcast %struct.point739* %0 to i8*
  %4 = bitcast %struct.point739* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo740(%struct.point740* noalias sret(%struct.point740) align 4 %0, %struct.point740* noundef byval(%struct.point740) align 4 %1) #0 {
  %3 = bitcast %struct.point740* %0 to i8*
  %4 = bitcast %struct.point740* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo741(%struct.point741* noalias sret(%struct.point741) align 4 %0, %struct.point741* noundef byval(%struct.point741) align 4 %1) #0 {
  %3 = bitcast %struct.point741* %0 to i8*
  %4 = bitcast %struct.point741* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo742(%struct.point742* noalias sret(%struct.point742) align 4 %0, %struct.point742* noundef byval(%struct.point742) align 4 %1) #0 {
  %3 = bitcast %struct.point742* %0 to i8*
  %4 = bitcast %struct.point742* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo743(%struct.point743* noalias sret(%struct.point743) align 4 %0, %struct.point743* noundef byval(%struct.point743) align 4 %1) #0 {
  %3 = bitcast %struct.point743* %0 to i8*
  %4 = bitcast %struct.point743* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo744(%struct.point744* noalias sret(%struct.point744) align 4 %0, %struct.point744* noundef byval(%struct.point744) align 4 %1) #0 {
  %3 = bitcast %struct.point744* %0 to i8*
  %4 = bitcast %struct.point744* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo745(%struct.point745* noalias sret(%struct.point745) align 4 %0, %struct.point745* noundef byval(%struct.point745) align 4 %1) #0 {
  %3 = bitcast %struct.point745* %0 to i8*
  %4 = bitcast %struct.point745* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo746(%struct.point746* noalias sret(%struct.point746) align 4 %0, %struct.point746* noundef byval(%struct.point746) align 4 %1) #0 {
  %3 = bitcast %struct.point746* %0 to i8*
  %4 = bitcast %struct.point746* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo747(%struct.point747* noalias sret(%struct.point747) align 4 %0, %struct.point747* noundef byval(%struct.point747) align 4 %1) #0 {
  %3 = bitcast %struct.point747* %0 to i8*
  %4 = bitcast %struct.point747* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo748(%struct.point748* noalias sret(%struct.point748) align 4 %0, %struct.point748* noundef byval(%struct.point748) align 4 %1) #0 {
  %3 = bitcast %struct.point748* %0 to i8*
  %4 = bitcast %struct.point748* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo749(%struct.point749* noalias sret(%struct.point749) align 4 %0, %struct.point749* noundef byval(%struct.point749) align 4 %1) #0 {
  %3 = bitcast %struct.point749* %0 to i8*
  %4 = bitcast %struct.point749* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo750(%struct.point750* noalias sret(%struct.point750) align 4 %0, %struct.point750* noundef byval(%struct.point750) align 4 %1) #0 {
  %3 = bitcast %struct.point750* %0 to i8*
  %4 = bitcast %struct.point750* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo751(%struct.point751* noalias sret(%struct.point751) align 4 %0, %struct.point751* noundef byval(%struct.point751) align 4 %1) #0 {
  %3 = bitcast %struct.point751* %0 to i8*
  %4 = bitcast %struct.point751* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo752(%struct.point752* noalias sret(%struct.point752) align 4 %0, %struct.point752* noundef byval(%struct.point752) align 4 %1) #0 {
  %3 = bitcast %struct.point752* %0 to i8*
  %4 = bitcast %struct.point752* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo753(%struct.point753* noalias sret(%struct.point753) align 4 %0, %struct.point753* noundef byval(%struct.point753) align 4 %1) #0 {
  %3 = bitcast %struct.point753* %0 to i8*
  %4 = bitcast %struct.point753* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo754(%struct.point754* noalias sret(%struct.point754) align 4 %0, %struct.point754* noundef byval(%struct.point754) align 4 %1) #0 {
  %3 = bitcast %struct.point754* %0 to i8*
  %4 = bitcast %struct.point754* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo755(%struct.point755* noalias sret(%struct.point755) align 4 %0, %struct.point755* noundef byval(%struct.point755) align 4 %1) #0 {
  %3 = bitcast %struct.point755* %0 to i8*
  %4 = bitcast %struct.point755* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo756(%struct.point756* noalias sret(%struct.point756) align 4 %0, %struct.point756* noundef byval(%struct.point756) align 4 %1) #0 {
  %3 = bitcast %struct.point756* %0 to i8*
  %4 = bitcast %struct.point756* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo757(%struct.point757* noalias sret(%struct.point757) align 4 %0, %struct.point757* noundef byval(%struct.point757) align 4 %1) #0 {
  %3 = bitcast %struct.point757* %0 to i8*
  %4 = bitcast %struct.point757* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo758(%struct.point758* noalias sret(%struct.point758) align 4 %0, %struct.point758* noundef byval(%struct.point758) align 4 %1) #0 {
  %3 = bitcast %struct.point758* %0 to i8*
  %4 = bitcast %struct.point758* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo759(%struct.point759* noalias sret(%struct.point759) align 4 %0, %struct.point759* noundef byval(%struct.point759) align 4 %1) #0 {
  %3 = bitcast %struct.point759* %0 to i8*
  %4 = bitcast %struct.point759* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo760(%struct.point760* noalias sret(%struct.point760) align 4 %0, %struct.point760* noundef byval(%struct.point760) align 4 %1) #0 {
  %3 = bitcast %struct.point760* %0 to i8*
  %4 = bitcast %struct.point760* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo761(%struct.point761* noalias sret(%struct.point761) align 4 %0, %struct.point761* noundef byval(%struct.point761) align 4 %1) #0 {
  %3 = bitcast %struct.point761* %0 to i8*
  %4 = bitcast %struct.point761* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo762(%struct.point762* noalias sret(%struct.point762) align 4 %0, %struct.point762* noundef byval(%struct.point762) align 4 %1) #0 {
  %3 = bitcast %struct.point762* %0 to i8*
  %4 = bitcast %struct.point762* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo763(%struct.point763* noalias sret(%struct.point763) align 4 %0, %struct.point763* noundef byval(%struct.point763) align 4 %1) #0 {
  %3 = bitcast %struct.point763* %0 to i8*
  %4 = bitcast %struct.point763* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo764(%struct.point764* noalias sret(%struct.point764) align 4 %0, %struct.point764* noundef byval(%struct.point764) align 4 %1) #0 {
  %3 = bitcast %struct.point764* %0 to i8*
  %4 = bitcast %struct.point764* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo765(%struct.point765* noalias sret(%struct.point765) align 4 %0, %struct.point765* noundef byval(%struct.point765) align 4 %1) #0 {
  %3 = bitcast %struct.point765* %0 to i8*
  %4 = bitcast %struct.point765* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo766(%struct.point766* noalias sret(%struct.point766) align 4 %0, %struct.point766* noundef byval(%struct.point766) align 4 %1) #0 {
  %3 = bitcast %struct.point766* %0 to i8*
  %4 = bitcast %struct.point766* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo767(%struct.point767* noalias sret(%struct.point767) align 4 %0, %struct.point767* noundef byval(%struct.point767) align 4 %1) #0 {
  %3 = bitcast %struct.point767* %0 to i8*
  %4 = bitcast %struct.point767* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo768(%struct.point768* noalias sret(%struct.point768) align 4 %0, %struct.point768* noundef byval(%struct.point768) align 4 %1) #0 {
  %3 = bitcast %struct.point768* %0 to i8*
  %4 = bitcast %struct.point768* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo769(%struct.point769* noalias sret(%struct.point769) align 4 %0, %struct.point769* noundef byval(%struct.point769) align 4 %1) #0 {
  %3 = bitcast %struct.point769* %0 to i8*
  %4 = bitcast %struct.point769* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo770(%struct.point770* noalias sret(%struct.point770) align 4 %0, %struct.point770* noundef byval(%struct.point770) align 4 %1) #0 {
  %3 = bitcast %struct.point770* %0 to i8*
  %4 = bitcast %struct.point770* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo771(%struct.point771* noalias sret(%struct.point771) align 4 %0, %struct.point771* noundef byval(%struct.point771) align 4 %1) #0 {
  %3 = bitcast %struct.point771* %0 to i8*
  %4 = bitcast %struct.point771* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo772(%struct.point772* noalias sret(%struct.point772) align 4 %0, %struct.point772* noundef byval(%struct.point772) align 4 %1) #0 {
  %3 = bitcast %struct.point772* %0 to i8*
  %4 = bitcast %struct.point772* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo773(%struct.point773* noalias sret(%struct.point773) align 4 %0, %struct.point773* noundef byval(%struct.point773) align 4 %1) #0 {
  %3 = bitcast %struct.point773* %0 to i8*
  %4 = bitcast %struct.point773* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo774(%struct.point774* noalias sret(%struct.point774) align 4 %0, %struct.point774* noundef byval(%struct.point774) align 4 %1) #0 {
  %3 = bitcast %struct.point774* %0 to i8*
  %4 = bitcast %struct.point774* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo775(%struct.point775* noalias sret(%struct.point775) align 4 %0, %struct.point775* noundef byval(%struct.point775) align 4 %1) #0 {
  %3 = bitcast %struct.point775* %0 to i8*
  %4 = bitcast %struct.point775* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo776(%struct.point776* noalias sret(%struct.point776) align 4 %0, %struct.point776* noundef byval(%struct.point776) align 4 %1) #0 {
  %3 = bitcast %struct.point776* %0 to i8*
  %4 = bitcast %struct.point776* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo777(%struct.point777* noalias sret(%struct.point777) align 4 %0, %struct.point777* noundef byval(%struct.point777) align 4 %1) #0 {
  %3 = bitcast %struct.point777* %0 to i8*
  %4 = bitcast %struct.point777* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo778(%struct.point778* noalias sret(%struct.point778) align 4 %0, %struct.point778* noundef byval(%struct.point778) align 4 %1) #0 {
  %3 = bitcast %struct.point778* %0 to i8*
  %4 = bitcast %struct.point778* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo779(%struct.point779* noalias sret(%struct.point779) align 4 %0, %struct.point779* noundef byval(%struct.point779) align 4 %1) #0 {
  %3 = bitcast %struct.point779* %0 to i8*
  %4 = bitcast %struct.point779* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo780(%struct.point780* noalias sret(%struct.point780) align 4 %0, %struct.point780* noundef byval(%struct.point780) align 4 %1) #0 {
  %3 = bitcast %struct.point780* %0 to i8*
  %4 = bitcast %struct.point780* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo781(%struct.point781* noalias sret(%struct.point781) align 4 %0, %struct.point781* noundef byval(%struct.point781) align 4 %1) #0 {
  %3 = bitcast %struct.point781* %0 to i8*
  %4 = bitcast %struct.point781* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo782(%struct.point782* noalias sret(%struct.point782) align 4 %0, %struct.point782* noundef byval(%struct.point782) align 4 %1) #0 {
  %3 = bitcast %struct.point782* %0 to i8*
  %4 = bitcast %struct.point782* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo783(%struct.point783* noalias sret(%struct.point783) align 4 %0, %struct.point783* noundef byval(%struct.point783) align 4 %1) #0 {
  %3 = bitcast %struct.point783* %0 to i8*
  %4 = bitcast %struct.point783* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo784(%struct.point784* noalias sret(%struct.point784) align 4 %0, %struct.point784* noundef byval(%struct.point784) align 4 %1) #0 {
  %3 = bitcast %struct.point784* %0 to i8*
  %4 = bitcast %struct.point784* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo785(%struct.point785* noalias sret(%struct.point785) align 4 %0, %struct.point785* noundef byval(%struct.point785) align 4 %1) #0 {
  %3 = bitcast %struct.point785* %0 to i8*
  %4 = bitcast %struct.point785* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo786(%struct.point786* noalias sret(%struct.point786) align 4 %0, %struct.point786* noundef byval(%struct.point786) align 4 %1) #0 {
  %3 = bitcast %struct.point786* %0 to i8*
  %4 = bitcast %struct.point786* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo787(%struct.point787* noalias sret(%struct.point787) align 4 %0, %struct.point787* noundef byval(%struct.point787) align 4 %1) #0 {
  %3 = bitcast %struct.point787* %0 to i8*
  %4 = bitcast %struct.point787* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo788(%struct.point788* noalias sret(%struct.point788) align 4 %0, %struct.point788* noundef byval(%struct.point788) align 4 %1) #0 {
  %3 = bitcast %struct.point788* %0 to i8*
  %4 = bitcast %struct.point788* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo789(%struct.point789* noalias sret(%struct.point789) align 4 %0, %struct.point789* noundef byval(%struct.point789) align 4 %1) #0 {
  %3 = bitcast %struct.point789* %0 to i8*
  %4 = bitcast %struct.point789* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo790(%struct.point790* noalias sret(%struct.point790) align 4 %0, %struct.point790* noundef byval(%struct.point790) align 4 %1) #0 {
  %3 = bitcast %struct.point790* %0 to i8*
  %4 = bitcast %struct.point790* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo791(%struct.point791* noalias sret(%struct.point791) align 4 %0, %struct.point791* noundef byval(%struct.point791) align 4 %1) #0 {
  %3 = bitcast %struct.point791* %0 to i8*
  %4 = bitcast %struct.point791* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo792(%struct.point792* noalias sret(%struct.point792) align 4 %0, %struct.point792* noundef byval(%struct.point792) align 4 %1) #0 {
  %3 = bitcast %struct.point792* %0 to i8*
  %4 = bitcast %struct.point792* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo793(%struct.point793* noalias sret(%struct.point793) align 4 %0, %struct.point793* noundef byval(%struct.point793) align 4 %1) #0 {
  %3 = bitcast %struct.point793* %0 to i8*
  %4 = bitcast %struct.point793* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo794(%struct.point794* noalias sret(%struct.point794) align 4 %0, %struct.point794* noundef byval(%struct.point794) align 4 %1) #0 {
  %3 = bitcast %struct.point794* %0 to i8*
  %4 = bitcast %struct.point794* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo795(%struct.point795* noalias sret(%struct.point795) align 4 %0, %struct.point795* noundef byval(%struct.point795) align 4 %1) #0 {
  %3 = bitcast %struct.point795* %0 to i8*
  %4 = bitcast %struct.point795* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo796(%struct.point796* noalias sret(%struct.point796) align 4 %0, %struct.point796* noundef byval(%struct.point796) align 4 %1) #0 {
  %3 = bitcast %struct.point796* %0 to i8*
  %4 = bitcast %struct.point796* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo797(%struct.point797* noalias sret(%struct.point797) align 4 %0, %struct.point797* noundef byval(%struct.point797) align 4 %1) #0 {
  %3 = bitcast %struct.point797* %0 to i8*
  %4 = bitcast %struct.point797* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo798(%struct.point798* noalias sret(%struct.point798) align 4 %0, %struct.point798* noundef byval(%struct.point798) align 4 %1) #0 {
  %3 = bitcast %struct.point798* %0 to i8*
  %4 = bitcast %struct.point798* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo799(%struct.point799* noalias sret(%struct.point799) align 4 %0, %struct.point799* noundef byval(%struct.point799) align 4 %1) #0 {
  %3 = bitcast %struct.point799* %0 to i8*
  %4 = bitcast %struct.point799* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo800(%struct.point800* noalias sret(%struct.point800) align 4 %0, %struct.point800* noundef byval(%struct.point800) align 4 %1) #0 {
  %3 = bitcast %struct.point800* %0 to i8*
  %4 = bitcast %struct.point800* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo801(%struct.point801* noalias sret(%struct.point801) align 4 %0, %struct.point801* noundef byval(%struct.point801) align 4 %1) #0 {
  %3 = bitcast %struct.point801* %0 to i8*
  %4 = bitcast %struct.point801* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo802(%struct.point802* noalias sret(%struct.point802) align 4 %0, %struct.point802* noundef byval(%struct.point802) align 4 %1) #0 {
  %3 = bitcast %struct.point802* %0 to i8*
  %4 = bitcast %struct.point802* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo803(%struct.point803* noalias sret(%struct.point803) align 4 %0, %struct.point803* noundef byval(%struct.point803) align 4 %1) #0 {
  %3 = bitcast %struct.point803* %0 to i8*
  %4 = bitcast %struct.point803* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo804(%struct.point804* noalias sret(%struct.point804) align 4 %0, %struct.point804* noundef byval(%struct.point804) align 4 %1) #0 {
  %3 = bitcast %struct.point804* %0 to i8*
  %4 = bitcast %struct.point804* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo805(%struct.point805* noalias sret(%struct.point805) align 4 %0, %struct.point805* noundef byval(%struct.point805) align 4 %1) #0 {
  %3 = bitcast %struct.point805* %0 to i8*
  %4 = bitcast %struct.point805* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo806(%struct.point806* noalias sret(%struct.point806) align 4 %0, %struct.point806* noundef byval(%struct.point806) align 4 %1) #0 {
  %3 = bitcast %struct.point806* %0 to i8*
  %4 = bitcast %struct.point806* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo807(%struct.point807* noalias sret(%struct.point807) align 4 %0, %struct.point807* noundef byval(%struct.point807) align 4 %1) #0 {
  %3 = bitcast %struct.point807* %0 to i8*
  %4 = bitcast %struct.point807* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo808(%struct.point808* noalias sret(%struct.point808) align 4 %0, %struct.point808* noundef byval(%struct.point808) align 4 %1) #0 {
  %3 = bitcast %struct.point808* %0 to i8*
  %4 = bitcast %struct.point808* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo809(%struct.point809* noalias sret(%struct.point809) align 4 %0, %struct.point809* noundef byval(%struct.point809) align 4 %1) #0 {
  %3 = bitcast %struct.point809* %0 to i8*
  %4 = bitcast %struct.point809* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo810(%struct.point810* noalias sret(%struct.point810) align 4 %0, %struct.point810* noundef byval(%struct.point810) align 4 %1) #0 {
  %3 = bitcast %struct.point810* %0 to i8*
  %4 = bitcast %struct.point810* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo811(%struct.point811* noalias sret(%struct.point811) align 4 %0, %struct.point811* noundef byval(%struct.point811) align 4 %1) #0 {
  %3 = bitcast %struct.point811* %0 to i8*
  %4 = bitcast %struct.point811* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo812(%struct.point812* noalias sret(%struct.point812) align 4 %0, %struct.point812* noundef byval(%struct.point812) align 4 %1) #0 {
  %3 = bitcast %struct.point812* %0 to i8*
  %4 = bitcast %struct.point812* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo813(%struct.point813* noalias sret(%struct.point813) align 4 %0, %struct.point813* noundef byval(%struct.point813) align 4 %1) #0 {
  %3 = bitcast %struct.point813* %0 to i8*
  %4 = bitcast %struct.point813* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo814(%struct.point814* noalias sret(%struct.point814) align 4 %0, %struct.point814* noundef byval(%struct.point814) align 4 %1) #0 {
  %3 = bitcast %struct.point814* %0 to i8*
  %4 = bitcast %struct.point814* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo815(%struct.point815* noalias sret(%struct.point815) align 4 %0, %struct.point815* noundef byval(%struct.point815) align 4 %1) #0 {
  %3 = bitcast %struct.point815* %0 to i8*
  %4 = bitcast %struct.point815* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo816(%struct.point816* noalias sret(%struct.point816) align 4 %0, %struct.point816* noundef byval(%struct.point816) align 4 %1) #0 {
  %3 = bitcast %struct.point816* %0 to i8*
  %4 = bitcast %struct.point816* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo817(%struct.point817* noalias sret(%struct.point817) align 4 %0, %struct.point817* noundef byval(%struct.point817) align 4 %1) #0 {
  %3 = bitcast %struct.point817* %0 to i8*
  %4 = bitcast %struct.point817* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo818(%struct.point818* noalias sret(%struct.point818) align 4 %0, %struct.point818* noundef byval(%struct.point818) align 4 %1) #0 {
  %3 = bitcast %struct.point818* %0 to i8*
  %4 = bitcast %struct.point818* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo819(%struct.point819* noalias sret(%struct.point819) align 4 %0, %struct.point819* noundef byval(%struct.point819) align 4 %1) #0 {
  %3 = bitcast %struct.point819* %0 to i8*
  %4 = bitcast %struct.point819* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo820(%struct.point820* noalias sret(%struct.point820) align 4 %0, %struct.point820* noundef byval(%struct.point820) align 4 %1) #0 {
  %3 = bitcast %struct.point820* %0 to i8*
  %4 = bitcast %struct.point820* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo821(%struct.point821* noalias sret(%struct.point821) align 4 %0, %struct.point821* noundef byval(%struct.point821) align 4 %1) #0 {
  %3 = bitcast %struct.point821* %0 to i8*
  %4 = bitcast %struct.point821* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo822(%struct.point822* noalias sret(%struct.point822) align 4 %0, %struct.point822* noundef byval(%struct.point822) align 4 %1) #0 {
  %3 = bitcast %struct.point822* %0 to i8*
  %4 = bitcast %struct.point822* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo823(%struct.point823* noalias sret(%struct.point823) align 4 %0, %struct.point823* noundef byval(%struct.point823) align 4 %1) #0 {
  %3 = bitcast %struct.point823* %0 to i8*
  %4 = bitcast %struct.point823* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo824(%struct.point824* noalias sret(%struct.point824) align 4 %0, %struct.point824* noundef byval(%struct.point824) align 4 %1) #0 {
  %3 = bitcast %struct.point824* %0 to i8*
  %4 = bitcast %struct.point824* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo825(%struct.point825* noalias sret(%struct.point825) align 4 %0, %struct.point825* noundef byval(%struct.point825) align 4 %1) #0 {
  %3 = bitcast %struct.point825* %0 to i8*
  %4 = bitcast %struct.point825* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo826(%struct.point826* noalias sret(%struct.point826) align 4 %0, %struct.point826* noundef byval(%struct.point826) align 4 %1) #0 {
  %3 = bitcast %struct.point826* %0 to i8*
  %4 = bitcast %struct.point826* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo827(%struct.point827* noalias sret(%struct.point827) align 4 %0, %struct.point827* noundef byval(%struct.point827) align 4 %1) #0 {
  %3 = bitcast %struct.point827* %0 to i8*
  %4 = bitcast %struct.point827* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo828(%struct.point828* noalias sret(%struct.point828) align 4 %0, %struct.point828* noundef byval(%struct.point828) align 4 %1) #0 {
  %3 = bitcast %struct.point828* %0 to i8*
  %4 = bitcast %struct.point828* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo829(%struct.point829* noalias sret(%struct.point829) align 4 %0, %struct.point829* noundef byval(%struct.point829) align 4 %1) #0 {
  %3 = bitcast %struct.point829* %0 to i8*
  %4 = bitcast %struct.point829* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo830(%struct.point830* noalias sret(%struct.point830) align 4 %0, %struct.point830* noundef byval(%struct.point830) align 4 %1) #0 {
  %3 = bitcast %struct.point830* %0 to i8*
  %4 = bitcast %struct.point830* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo831(%struct.point831* noalias sret(%struct.point831) align 4 %0, %struct.point831* noundef byval(%struct.point831) align 4 %1) #0 {
  %3 = bitcast %struct.point831* %0 to i8*
  %4 = bitcast %struct.point831* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo832(%struct.point832* noalias sret(%struct.point832) align 4 %0, %struct.point832* noundef byval(%struct.point832) align 4 %1) #0 {
  %3 = bitcast %struct.point832* %0 to i8*
  %4 = bitcast %struct.point832* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo833(%struct.point833* noalias sret(%struct.point833) align 4 %0, %struct.point833* noundef byval(%struct.point833) align 4 %1) #0 {
  %3 = bitcast %struct.point833* %0 to i8*
  %4 = bitcast %struct.point833* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo834(%struct.point834* noalias sret(%struct.point834) align 4 %0, %struct.point834* noundef byval(%struct.point834) align 4 %1) #0 {
  %3 = bitcast %struct.point834* %0 to i8*
  %4 = bitcast %struct.point834* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo835(%struct.point835* noalias sret(%struct.point835) align 4 %0, %struct.point835* noundef byval(%struct.point835) align 4 %1) #0 {
  %3 = bitcast %struct.point835* %0 to i8*
  %4 = bitcast %struct.point835* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo836(%struct.point836* noalias sret(%struct.point836) align 4 %0, %struct.point836* noundef byval(%struct.point836) align 4 %1) #0 {
  %3 = bitcast %struct.point836* %0 to i8*
  %4 = bitcast %struct.point836* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo837(%struct.point837* noalias sret(%struct.point837) align 4 %0, %struct.point837* noundef byval(%struct.point837) align 4 %1) #0 {
  %3 = bitcast %struct.point837* %0 to i8*
  %4 = bitcast %struct.point837* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo838(%struct.point838* noalias sret(%struct.point838) align 4 %0, %struct.point838* noundef byval(%struct.point838) align 4 %1) #0 {
  %3 = bitcast %struct.point838* %0 to i8*
  %4 = bitcast %struct.point838* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo839(%struct.point839* noalias sret(%struct.point839) align 4 %0, %struct.point839* noundef byval(%struct.point839) align 4 %1) #0 {
  %3 = bitcast %struct.point839* %0 to i8*
  %4 = bitcast %struct.point839* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo840(%struct.point840* noalias sret(%struct.point840) align 4 %0, %struct.point840* noundef byval(%struct.point840) align 4 %1) #0 {
  %3 = bitcast %struct.point840* %0 to i8*
  %4 = bitcast %struct.point840* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo841(%struct.point841* noalias sret(%struct.point841) align 4 %0, %struct.point841* noundef byval(%struct.point841) align 4 %1) #0 {
  %3 = bitcast %struct.point841* %0 to i8*
  %4 = bitcast %struct.point841* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo842(%struct.point842* noalias sret(%struct.point842) align 4 %0, %struct.point842* noundef byval(%struct.point842) align 4 %1) #0 {
  %3 = bitcast %struct.point842* %0 to i8*
  %4 = bitcast %struct.point842* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo843(%struct.point843* noalias sret(%struct.point843) align 4 %0, %struct.point843* noundef byval(%struct.point843) align 4 %1) #0 {
  %3 = bitcast %struct.point843* %0 to i8*
  %4 = bitcast %struct.point843* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo844(%struct.point844* noalias sret(%struct.point844) align 4 %0, %struct.point844* noundef byval(%struct.point844) align 4 %1) #0 {
  %3 = bitcast %struct.point844* %0 to i8*
  %4 = bitcast %struct.point844* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo845(%struct.point845* noalias sret(%struct.point845) align 4 %0, %struct.point845* noundef byval(%struct.point845) align 4 %1) #0 {
  %3 = bitcast %struct.point845* %0 to i8*
  %4 = bitcast %struct.point845* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo846(%struct.point846* noalias sret(%struct.point846) align 4 %0, %struct.point846* noundef byval(%struct.point846) align 4 %1) #0 {
  %3 = bitcast %struct.point846* %0 to i8*
  %4 = bitcast %struct.point846* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo847(%struct.point847* noalias sret(%struct.point847) align 4 %0, %struct.point847* noundef byval(%struct.point847) align 4 %1) #0 {
  %3 = bitcast %struct.point847* %0 to i8*
  %4 = bitcast %struct.point847* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo848(%struct.point848* noalias sret(%struct.point848) align 4 %0, %struct.point848* noundef byval(%struct.point848) align 4 %1) #0 {
  %3 = bitcast %struct.point848* %0 to i8*
  %4 = bitcast %struct.point848* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo849(%struct.point849* noalias sret(%struct.point849) align 4 %0, %struct.point849* noundef byval(%struct.point849) align 4 %1) #0 {
  %3 = bitcast %struct.point849* %0 to i8*
  %4 = bitcast %struct.point849* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo850(%struct.point850* noalias sret(%struct.point850) align 4 %0, %struct.point850* noundef byval(%struct.point850) align 4 %1) #0 {
  %3 = bitcast %struct.point850* %0 to i8*
  %4 = bitcast %struct.point850* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo851(%struct.point851* noalias sret(%struct.point851) align 4 %0, %struct.point851* noundef byval(%struct.point851) align 4 %1) #0 {
  %3 = bitcast %struct.point851* %0 to i8*
  %4 = bitcast %struct.point851* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo852(%struct.point852* noalias sret(%struct.point852) align 4 %0, %struct.point852* noundef byval(%struct.point852) align 4 %1) #0 {
  %3 = bitcast %struct.point852* %0 to i8*
  %4 = bitcast %struct.point852* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo853(%struct.point853* noalias sret(%struct.point853) align 4 %0, %struct.point853* noundef byval(%struct.point853) align 4 %1) #0 {
  %3 = bitcast %struct.point853* %0 to i8*
  %4 = bitcast %struct.point853* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo854(%struct.point854* noalias sret(%struct.point854) align 4 %0, %struct.point854* noundef byval(%struct.point854) align 4 %1) #0 {
  %3 = bitcast %struct.point854* %0 to i8*
  %4 = bitcast %struct.point854* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo855(%struct.point855* noalias sret(%struct.point855) align 4 %0, %struct.point855* noundef byval(%struct.point855) align 4 %1) #0 {
  %3 = bitcast %struct.point855* %0 to i8*
  %4 = bitcast %struct.point855* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo856(%struct.point856* noalias sret(%struct.point856) align 4 %0, %struct.point856* noundef byval(%struct.point856) align 4 %1) #0 {
  %3 = bitcast %struct.point856* %0 to i8*
  %4 = bitcast %struct.point856* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo857(%struct.point857* noalias sret(%struct.point857) align 4 %0, %struct.point857* noundef byval(%struct.point857) align 4 %1) #0 {
  %3 = bitcast %struct.point857* %0 to i8*
  %4 = bitcast %struct.point857* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo858(%struct.point858* noalias sret(%struct.point858) align 4 %0, %struct.point858* noundef byval(%struct.point858) align 4 %1) #0 {
  %3 = bitcast %struct.point858* %0 to i8*
  %4 = bitcast %struct.point858* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo859(%struct.point859* noalias sret(%struct.point859) align 4 %0, %struct.point859* noundef byval(%struct.point859) align 4 %1) #0 {
  %3 = bitcast %struct.point859* %0 to i8*
  %4 = bitcast %struct.point859* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo860(%struct.point860* noalias sret(%struct.point860) align 4 %0, %struct.point860* noundef byval(%struct.point860) align 4 %1) #0 {
  %3 = bitcast %struct.point860* %0 to i8*
  %4 = bitcast %struct.point860* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo861(%struct.point861* noalias sret(%struct.point861) align 4 %0, %struct.point861* noundef byval(%struct.point861) align 4 %1) #0 {
  %3 = bitcast %struct.point861* %0 to i8*
  %4 = bitcast %struct.point861* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo862(%struct.point862* noalias sret(%struct.point862) align 4 %0, %struct.point862* noundef byval(%struct.point862) align 4 %1) #0 {
  %3 = bitcast %struct.point862* %0 to i8*
  %4 = bitcast %struct.point862* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo863(%struct.point863* noalias sret(%struct.point863) align 4 %0, %struct.point863* noundef byval(%struct.point863) align 4 %1) #0 {
  %3 = bitcast %struct.point863* %0 to i8*
  %4 = bitcast %struct.point863* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo864(%struct.point864* noalias sret(%struct.point864) align 4 %0, %struct.point864* noundef byval(%struct.point864) align 4 %1) #0 {
  %3 = bitcast %struct.point864* %0 to i8*
  %4 = bitcast %struct.point864* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo865(%struct.point865* noalias sret(%struct.point865) align 4 %0, %struct.point865* noundef byval(%struct.point865) align 4 %1) #0 {
  %3 = bitcast %struct.point865* %0 to i8*
  %4 = bitcast %struct.point865* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo866(%struct.point866* noalias sret(%struct.point866) align 4 %0, %struct.point866* noundef byval(%struct.point866) align 4 %1) #0 {
  %3 = bitcast %struct.point866* %0 to i8*
  %4 = bitcast %struct.point866* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo867(%struct.point867* noalias sret(%struct.point867) align 4 %0, %struct.point867* noundef byval(%struct.point867) align 4 %1) #0 {
  %3 = bitcast %struct.point867* %0 to i8*
  %4 = bitcast %struct.point867* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo868(%struct.point868* noalias sret(%struct.point868) align 4 %0, %struct.point868* noundef byval(%struct.point868) align 4 %1) #0 {
  %3 = bitcast %struct.point868* %0 to i8*
  %4 = bitcast %struct.point868* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo869(%struct.point869* noalias sret(%struct.point869) align 4 %0, %struct.point869* noundef byval(%struct.point869) align 4 %1) #0 {
  %3 = bitcast %struct.point869* %0 to i8*
  %4 = bitcast %struct.point869* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo870(%struct.point870* noalias sret(%struct.point870) align 4 %0, %struct.point870* noundef byval(%struct.point870) align 4 %1) #0 {
  %3 = bitcast %struct.point870* %0 to i8*
  %4 = bitcast %struct.point870* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo871(%struct.point871* noalias sret(%struct.point871) align 4 %0, %struct.point871* noundef byval(%struct.point871) align 4 %1) #0 {
  %3 = bitcast %struct.point871* %0 to i8*
  %4 = bitcast %struct.point871* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo872(%struct.point872* noalias sret(%struct.point872) align 4 %0, %struct.point872* noundef byval(%struct.point872) align 4 %1) #0 {
  %3 = bitcast %struct.point872* %0 to i8*
  %4 = bitcast %struct.point872* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo873(%struct.point873* noalias sret(%struct.point873) align 4 %0, %struct.point873* noundef byval(%struct.point873) align 4 %1) #0 {
  %3 = bitcast %struct.point873* %0 to i8*
  %4 = bitcast %struct.point873* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo874(%struct.point874* noalias sret(%struct.point874) align 4 %0, %struct.point874* noundef byval(%struct.point874) align 4 %1) #0 {
  %3 = bitcast %struct.point874* %0 to i8*
  %4 = bitcast %struct.point874* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo875(%struct.point875* noalias sret(%struct.point875) align 4 %0, %struct.point875* noundef byval(%struct.point875) align 4 %1) #0 {
  %3 = bitcast %struct.point875* %0 to i8*
  %4 = bitcast %struct.point875* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo876(%struct.point876* noalias sret(%struct.point876) align 4 %0, %struct.point876* noundef byval(%struct.point876) align 4 %1) #0 {
  %3 = bitcast %struct.point876* %0 to i8*
  %4 = bitcast %struct.point876* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo877(%struct.point877* noalias sret(%struct.point877) align 4 %0, %struct.point877* noundef byval(%struct.point877) align 4 %1) #0 {
  %3 = bitcast %struct.point877* %0 to i8*
  %4 = bitcast %struct.point877* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo878(%struct.point878* noalias sret(%struct.point878) align 4 %0, %struct.point878* noundef byval(%struct.point878) align 4 %1) #0 {
  %3 = bitcast %struct.point878* %0 to i8*
  %4 = bitcast %struct.point878* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo879(%struct.point879* noalias sret(%struct.point879) align 4 %0, %struct.point879* noundef byval(%struct.point879) align 4 %1) #0 {
  %3 = bitcast %struct.point879* %0 to i8*
  %4 = bitcast %struct.point879* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo880(%struct.point880* noalias sret(%struct.point880) align 4 %0, %struct.point880* noundef byval(%struct.point880) align 4 %1) #0 {
  %3 = bitcast %struct.point880* %0 to i8*
  %4 = bitcast %struct.point880* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo881(%struct.point881* noalias sret(%struct.point881) align 4 %0, %struct.point881* noundef byval(%struct.point881) align 4 %1) #0 {
  %3 = bitcast %struct.point881* %0 to i8*
  %4 = bitcast %struct.point881* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo882(%struct.point882* noalias sret(%struct.point882) align 4 %0, %struct.point882* noundef byval(%struct.point882) align 4 %1) #0 {
  %3 = bitcast %struct.point882* %0 to i8*
  %4 = bitcast %struct.point882* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo883(%struct.point883* noalias sret(%struct.point883) align 4 %0, %struct.point883* noundef byval(%struct.point883) align 4 %1) #0 {
  %3 = bitcast %struct.point883* %0 to i8*
  %4 = bitcast %struct.point883* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo884(%struct.point884* noalias sret(%struct.point884) align 4 %0, %struct.point884* noundef byval(%struct.point884) align 4 %1) #0 {
  %3 = bitcast %struct.point884* %0 to i8*
  %4 = bitcast %struct.point884* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo885(%struct.point885* noalias sret(%struct.point885) align 4 %0, %struct.point885* noundef byval(%struct.point885) align 4 %1) #0 {
  %3 = bitcast %struct.point885* %0 to i8*
  %4 = bitcast %struct.point885* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo886(%struct.point886* noalias sret(%struct.point886) align 4 %0, %struct.point886* noundef byval(%struct.point886) align 4 %1) #0 {
  %3 = bitcast %struct.point886* %0 to i8*
  %4 = bitcast %struct.point886* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo887(%struct.point887* noalias sret(%struct.point887) align 4 %0, %struct.point887* noundef byval(%struct.point887) align 4 %1) #0 {
  %3 = bitcast %struct.point887* %0 to i8*
  %4 = bitcast %struct.point887* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo888(%struct.point888* noalias sret(%struct.point888) align 4 %0, %struct.point888* noundef byval(%struct.point888) align 4 %1) #0 {
  %3 = bitcast %struct.point888* %0 to i8*
  %4 = bitcast %struct.point888* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo889(%struct.point889* noalias sret(%struct.point889) align 4 %0, %struct.point889* noundef byval(%struct.point889) align 4 %1) #0 {
  %3 = bitcast %struct.point889* %0 to i8*
  %4 = bitcast %struct.point889* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo890(%struct.point890* noalias sret(%struct.point890) align 4 %0, %struct.point890* noundef byval(%struct.point890) align 4 %1) #0 {
  %3 = bitcast %struct.point890* %0 to i8*
  %4 = bitcast %struct.point890* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo891(%struct.point891* noalias sret(%struct.point891) align 4 %0, %struct.point891* noundef byval(%struct.point891) align 4 %1) #0 {
  %3 = bitcast %struct.point891* %0 to i8*
  %4 = bitcast %struct.point891* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo892(%struct.point892* noalias sret(%struct.point892) align 4 %0, %struct.point892* noundef byval(%struct.point892) align 4 %1) #0 {
  %3 = bitcast %struct.point892* %0 to i8*
  %4 = bitcast %struct.point892* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo893(%struct.point893* noalias sret(%struct.point893) align 4 %0, %struct.point893* noundef byval(%struct.point893) align 4 %1) #0 {
  %3 = bitcast %struct.point893* %0 to i8*
  %4 = bitcast %struct.point893* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo894(%struct.point894* noalias sret(%struct.point894) align 4 %0, %struct.point894* noundef byval(%struct.point894) align 4 %1) #0 {
  %3 = bitcast %struct.point894* %0 to i8*
  %4 = bitcast %struct.point894* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo895(%struct.point895* noalias sret(%struct.point895) align 4 %0, %struct.point895* noundef byval(%struct.point895) align 4 %1) #0 {
  %3 = bitcast %struct.point895* %0 to i8*
  %4 = bitcast %struct.point895* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo896(%struct.point896* noalias sret(%struct.point896) align 4 %0, %struct.point896* noundef byval(%struct.point896) align 4 %1) #0 {
  %3 = bitcast %struct.point896* %0 to i8*
  %4 = bitcast %struct.point896* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo897(%struct.point897* noalias sret(%struct.point897) align 4 %0, %struct.point897* noundef byval(%struct.point897) align 4 %1) #0 {
  %3 = bitcast %struct.point897* %0 to i8*
  %4 = bitcast %struct.point897* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo898(%struct.point898* noalias sret(%struct.point898) align 4 %0, %struct.point898* noundef byval(%struct.point898) align 4 %1) #0 {
  %3 = bitcast %struct.point898* %0 to i8*
  %4 = bitcast %struct.point898* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo899(%struct.point899* noalias sret(%struct.point899) align 4 %0, %struct.point899* noundef byval(%struct.point899) align 4 %1) #0 {
  %3 = bitcast %struct.point899* %0 to i8*
  %4 = bitcast %struct.point899* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo900(%struct.point900* noalias sret(%struct.point900) align 4 %0, %struct.point900* noundef byval(%struct.point900) align 4 %1) #0 {
  %3 = bitcast %struct.point900* %0 to i8*
  %4 = bitcast %struct.point900* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo901(%struct.point901* noalias sret(%struct.point901) align 4 %0, %struct.point901* noundef byval(%struct.point901) align 4 %1) #0 {
  %3 = bitcast %struct.point901* %0 to i8*
  %4 = bitcast %struct.point901* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo902(%struct.point902* noalias sret(%struct.point902) align 4 %0, %struct.point902* noundef byval(%struct.point902) align 4 %1) #0 {
  %3 = bitcast %struct.point902* %0 to i8*
  %4 = bitcast %struct.point902* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo903(%struct.point903* noalias sret(%struct.point903) align 4 %0, %struct.point903* noundef byval(%struct.point903) align 4 %1) #0 {
  %3 = bitcast %struct.point903* %0 to i8*
  %4 = bitcast %struct.point903* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo904(%struct.point904* noalias sret(%struct.point904) align 4 %0, %struct.point904* noundef byval(%struct.point904) align 4 %1) #0 {
  %3 = bitcast %struct.point904* %0 to i8*
  %4 = bitcast %struct.point904* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 8, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo905(%struct.point905* noalias sret(%struct.point905) align 4 %0, %struct.point905* noundef byval(%struct.point905) align 4 %1) #0 {
  %3 = bitcast %struct.point905* %0 to i8*
  %4 = bitcast %struct.point905* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo906(%struct.point906* noalias sret(%struct.point906) align 4 %0, %struct.point906* noundef byval(%struct.point906) align 4 %1) #0 {
  %3 = bitcast %struct.point906* %0 to i8*
  %4 = bitcast %struct.point906* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo907(%struct.point907* noalias sret(%struct.point907) align 4 %0, %struct.point907* noundef byval(%struct.point907) align 4 %1) #0 {
  %3 = bitcast %struct.point907* %0 to i8*
  %4 = bitcast %struct.point907* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo908(%struct.point908* noalias sret(%struct.point908) align 4 %0, %struct.point908* noundef byval(%struct.point908) align 4 %1) #0 {
  %3 = bitcast %struct.point908* %0 to i8*
  %4 = bitcast %struct.point908* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo909(%struct.point909* noalias sret(%struct.point909) align 4 %0, %struct.point909* noundef byval(%struct.point909) align 4 %1) #0 {
  %3 = bitcast %struct.point909* %0 to i8*
  %4 = bitcast %struct.point909* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo910(%struct.point910* noalias sret(%struct.point910) align 4 %0, %struct.point910* noundef byval(%struct.point910) align 4 %1) #0 {
  %3 = bitcast %struct.point910* %0 to i8*
  %4 = bitcast %struct.point910* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo911(%struct.point911* noalias sret(%struct.point911) align 4 %0, %struct.point911* noundef byval(%struct.point911) align 4 %1) #0 {
  %3 = bitcast %struct.point911* %0 to i8*
  %4 = bitcast %struct.point911* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo912(%struct.point912* noalias sret(%struct.point912) align 4 %0, %struct.point912* noundef byval(%struct.point912) align 4 %1) #0 {
  %3 = bitcast %struct.point912* %0 to i8*
  %4 = bitcast %struct.point912* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo913(%struct.point913* noalias sret(%struct.point913) align 4 %0, %struct.point913* noundef byval(%struct.point913) align 4 %1) #0 {
  %3 = bitcast %struct.point913* %0 to i8*
  %4 = bitcast %struct.point913* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo914(%struct.point914* noalias sret(%struct.point914) align 4 %0, %struct.point914* noundef byval(%struct.point914) align 4 %1) #0 {
  %3 = bitcast %struct.point914* %0 to i8*
  %4 = bitcast %struct.point914* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo915(%struct.point915* noalias sret(%struct.point915) align 4 %0, %struct.point915* noundef byval(%struct.point915) align 4 %1) #0 {
  %3 = bitcast %struct.point915* %0 to i8*
  %4 = bitcast %struct.point915* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo916(%struct.point916* noalias sret(%struct.point916) align 4 %0, %struct.point916* noundef byval(%struct.point916) align 4 %1) #0 {
  %3 = bitcast %struct.point916* %0 to i8*
  %4 = bitcast %struct.point916* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo917(%struct.point917* noalias sret(%struct.point917) align 4 %0, %struct.point917* noundef byval(%struct.point917) align 4 %1) #0 {
  %3 = bitcast %struct.point917* %0 to i8*
  %4 = bitcast %struct.point917* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo918(%struct.point918* noalias sret(%struct.point918) align 4 %0, %struct.point918* noundef byval(%struct.point918) align 4 %1) #0 {
  %3 = bitcast %struct.point918* %0 to i8*
  %4 = bitcast %struct.point918* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo919(%struct.point919* noalias sret(%struct.point919) align 4 %0, %struct.point919* noundef byval(%struct.point919) align 4 %1) #0 {
  %3 = bitcast %struct.point919* %0 to i8*
  %4 = bitcast %struct.point919* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo920(%struct.point920* noalias sret(%struct.point920) align 4 %0, %struct.point920* noundef byval(%struct.point920) align 4 %1) #0 {
  %3 = bitcast %struct.point920* %0 to i8*
  %4 = bitcast %struct.point920* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo921(%struct.point921* noalias sret(%struct.point921) align 4 %0, %struct.point921* noundef byval(%struct.point921) align 4 %1) #0 {
  %3 = bitcast %struct.point921* %0 to i8*
  %4 = bitcast %struct.point921* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo922(%struct.point922* noalias sret(%struct.point922) align 4 %0, %struct.point922* noundef byval(%struct.point922) align 4 %1) #0 {
  %3 = bitcast %struct.point922* %0 to i8*
  %4 = bitcast %struct.point922* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo923(%struct.point923* noalias sret(%struct.point923) align 4 %0, %struct.point923* noundef byval(%struct.point923) align 4 %1) #0 {
  %3 = bitcast %struct.point923* %0 to i8*
  %4 = bitcast %struct.point923* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo924(%struct.point924* noalias sret(%struct.point924) align 4 %0, %struct.point924* noundef byval(%struct.point924) align 4 %1) #0 {
  %3 = bitcast %struct.point924* %0 to i8*
  %4 = bitcast %struct.point924* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo925(%struct.point925* noalias sret(%struct.point925) align 4 %0, %struct.point925* noundef byval(%struct.point925) align 4 %1) #0 {
  %3 = bitcast %struct.point925* %0 to i8*
  %4 = bitcast %struct.point925* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo926(%struct.point926* noalias sret(%struct.point926) align 4 %0, %struct.point926* noundef byval(%struct.point926) align 4 %1) #0 {
  %3 = bitcast %struct.point926* %0 to i8*
  %4 = bitcast %struct.point926* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo927(%struct.point927* noalias sret(%struct.point927) align 4 %0, %struct.point927* noundef byval(%struct.point927) align 4 %1) #0 {
  %3 = bitcast %struct.point927* %0 to i8*
  %4 = bitcast %struct.point927* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo928(%struct.point928* noalias sret(%struct.point928) align 4 %0, %struct.point928* noundef byval(%struct.point928) align 4 %1) #0 {
  %3 = bitcast %struct.point928* %0 to i8*
  %4 = bitcast %struct.point928* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo929(%struct.point929* noalias sret(%struct.point929) align 4 %0, %struct.point929* noundef byval(%struct.point929) align 4 %1) #0 {
  %3 = bitcast %struct.point929* %0 to i8*
  %4 = bitcast %struct.point929* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo930(%struct.point930* noalias sret(%struct.point930) align 4 %0, %struct.point930* noundef byval(%struct.point930) align 4 %1) #0 {
  %3 = bitcast %struct.point930* %0 to i8*
  %4 = bitcast %struct.point930* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo931(%struct.point931* noalias sret(%struct.point931) align 4 %0, %struct.point931* noundef byval(%struct.point931) align 4 %1) #0 {
  %3 = bitcast %struct.point931* %0 to i8*
  %4 = bitcast %struct.point931* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo932(%struct.point932* noalias sret(%struct.point932) align 4 %0, %struct.point932* noundef byval(%struct.point932) align 4 %1) #0 {
  %3 = bitcast %struct.point932* %0 to i8*
  %4 = bitcast %struct.point932* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo933(%struct.point933* noalias sret(%struct.point933) align 4 %0, %struct.point933* noundef byval(%struct.point933) align 4 %1) #0 {
  %3 = bitcast %struct.point933* %0 to i8*
  %4 = bitcast %struct.point933* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo934(%struct.point934* noalias sret(%struct.point934) align 4 %0, %struct.point934* noundef byval(%struct.point934) align 4 %1) #0 {
  %3 = bitcast %struct.point934* %0 to i8*
  %4 = bitcast %struct.point934* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo935(%struct.point935* noalias sret(%struct.point935) align 4 %0, %struct.point935* noundef byval(%struct.point935) align 4 %1) #0 {
  %3 = bitcast %struct.point935* %0 to i8*
  %4 = bitcast %struct.point935* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo936(%struct.point936* noalias sret(%struct.point936) align 4 %0, %struct.point936* noundef byval(%struct.point936) align 4 %1) #0 {
  %3 = bitcast %struct.point936* %0 to i8*
  %4 = bitcast %struct.point936* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo937(%struct.point937* noalias sret(%struct.point937) align 4 %0, %struct.point937* noundef byval(%struct.point937) align 4 %1) #0 {
  %3 = bitcast %struct.point937* %0 to i8*
  %4 = bitcast %struct.point937* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo938(%struct.point938* noalias sret(%struct.point938) align 4 %0, %struct.point938* noundef byval(%struct.point938) align 4 %1) #0 {
  %3 = bitcast %struct.point938* %0 to i8*
  %4 = bitcast %struct.point938* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo939(%struct.point939* noalias sret(%struct.point939) align 4 %0, %struct.point939* noundef byval(%struct.point939) align 4 %1) #0 {
  %3 = bitcast %struct.point939* %0 to i8*
  %4 = bitcast %struct.point939* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo940(%struct.point940* noalias sret(%struct.point940) align 4 %0, %struct.point940* noundef byval(%struct.point940) align 4 %1) #0 {
  %3 = bitcast %struct.point940* %0 to i8*
  %4 = bitcast %struct.point940* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo941(%struct.point941* noalias sret(%struct.point941) align 4 %0, %struct.point941* noundef byval(%struct.point941) align 4 %1) #0 {
  %3 = bitcast %struct.point941* %0 to i8*
  %4 = bitcast %struct.point941* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo942(%struct.point942* noalias sret(%struct.point942) align 4 %0, %struct.point942* noundef byval(%struct.point942) align 4 %1) #0 {
  %3 = bitcast %struct.point942* %0 to i8*
  %4 = bitcast %struct.point942* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo943(%struct.point943* noalias sret(%struct.point943) align 4 %0, %struct.point943* noundef byval(%struct.point943) align 4 %1) #0 {
  %3 = bitcast %struct.point943* %0 to i8*
  %4 = bitcast %struct.point943* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo944(%struct.point944* noalias sret(%struct.point944) align 4 %0, %struct.point944* noundef byval(%struct.point944) align 4 %1) #0 {
  %3 = bitcast %struct.point944* %0 to i8*
  %4 = bitcast %struct.point944* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo945(%struct.point945* noalias sret(%struct.point945) align 4 %0, %struct.point945* noundef byval(%struct.point945) align 4 %1) #0 {
  %3 = bitcast %struct.point945* %0 to i8*
  %4 = bitcast %struct.point945* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo946(%struct.point946* noalias sret(%struct.point946) align 4 %0, %struct.point946* noundef byval(%struct.point946) align 4 %1) #0 {
  %3 = bitcast %struct.point946* %0 to i8*
  %4 = bitcast %struct.point946* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo947(%struct.point947* noalias sret(%struct.point947) align 4 %0, %struct.point947* noundef byval(%struct.point947) align 4 %1) #0 {
  %3 = bitcast %struct.point947* %0 to i8*
  %4 = bitcast %struct.point947* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo948(%struct.point948* noalias sret(%struct.point948) align 4 %0, %struct.point948* noundef byval(%struct.point948) align 4 %1) #0 {
  %3 = bitcast %struct.point948* %0 to i8*
  %4 = bitcast %struct.point948* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo949(%struct.point949* noalias sret(%struct.point949) align 4 %0, %struct.point949* noundef byval(%struct.point949) align 4 %1) #0 {
  %3 = bitcast %struct.point949* %0 to i8*
  %4 = bitcast %struct.point949* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo950(%struct.point950* noalias sret(%struct.point950) align 4 %0, %struct.point950* noundef byval(%struct.point950) align 4 %1) #0 {
  %3 = bitcast %struct.point950* %0 to i8*
  %4 = bitcast %struct.point950* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo951(%struct.point951* noalias sret(%struct.point951) align 4 %0, %struct.point951* noundef byval(%struct.point951) align 4 %1) #0 {
  %3 = bitcast %struct.point951* %0 to i8*
  %4 = bitcast %struct.point951* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo952(%struct.point952* noalias sret(%struct.point952) align 4 %0, %struct.point952* noundef byval(%struct.point952) align 4 %1) #0 {
  %3 = bitcast %struct.point952* %0 to i8*
  %4 = bitcast %struct.point952* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo953(%struct.point953* noalias sret(%struct.point953) align 4 %0, %struct.point953* noundef byval(%struct.point953) align 4 %1) #0 {
  %3 = bitcast %struct.point953* %0 to i8*
  %4 = bitcast %struct.point953* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo954(%struct.point954* noalias sret(%struct.point954) align 4 %0, %struct.point954* noundef byval(%struct.point954) align 4 %1) #0 {
  %3 = bitcast %struct.point954* %0 to i8*
  %4 = bitcast %struct.point954* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo955(%struct.point955* noalias sret(%struct.point955) align 4 %0, %struct.point955* noundef byval(%struct.point955) align 4 %1) #0 {
  %3 = bitcast %struct.point955* %0 to i8*
  %4 = bitcast %struct.point955* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo956(%struct.point956* noalias sret(%struct.point956) align 4 %0, %struct.point956* noundef byval(%struct.point956) align 4 %1) #0 {
  %3 = bitcast %struct.point956* %0 to i8*
  %4 = bitcast %struct.point956* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo957(%struct.point957* noalias sret(%struct.point957) align 4 %0, %struct.point957* noundef byval(%struct.point957) align 4 %1) #0 {
  %3 = bitcast %struct.point957* %0 to i8*
  %4 = bitcast %struct.point957* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo958(%struct.point958* noalias sret(%struct.point958) align 4 %0, %struct.point958* noundef byval(%struct.point958) align 4 %1) #0 {
  %3 = bitcast %struct.point958* %0 to i8*
  %4 = bitcast %struct.point958* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo959(%struct.point959* noalias sret(%struct.point959) align 4 %0, %struct.point959* noundef byval(%struct.point959) align 4 %1) #0 {
  %3 = bitcast %struct.point959* %0 to i8*
  %4 = bitcast %struct.point959* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo960(%struct.point960* noalias sret(%struct.point960) align 4 %0, %struct.point960* noundef byval(%struct.point960) align 4 %1) #0 {
  %3 = bitcast %struct.point960* %0 to i8*
  %4 = bitcast %struct.point960* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo961(%struct.point961* noalias sret(%struct.point961) align 4 %0, %struct.point961* noundef byval(%struct.point961) align 4 %1) #0 {
  %3 = bitcast %struct.point961* %0 to i8*
  %4 = bitcast %struct.point961* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo962(%struct.point962* noalias sret(%struct.point962) align 4 %0, %struct.point962* noundef byval(%struct.point962) align 4 %1) #0 {
  %3 = bitcast %struct.point962* %0 to i8*
  %4 = bitcast %struct.point962* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo963(%struct.point963* noalias sret(%struct.point963) align 4 %0, %struct.point963* noundef byval(%struct.point963) align 4 %1) #0 {
  %3 = bitcast %struct.point963* %0 to i8*
  %4 = bitcast %struct.point963* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo964(%struct.point964* noalias sret(%struct.point964) align 4 %0, %struct.point964* noundef byval(%struct.point964) align 4 %1) #0 {
  %3 = bitcast %struct.point964* %0 to i8*
  %4 = bitcast %struct.point964* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo965(%struct.point965* noalias sret(%struct.point965) align 4 %0, %struct.point965* noundef byval(%struct.point965) align 4 %1) #0 {
  %3 = bitcast %struct.point965* %0 to i8*
  %4 = bitcast %struct.point965* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo966(%struct.point966* noalias sret(%struct.point966) align 4 %0, %struct.point966* noundef byval(%struct.point966) align 4 %1) #0 {
  %3 = bitcast %struct.point966* %0 to i8*
  %4 = bitcast %struct.point966* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo967(%struct.point967* noalias sret(%struct.point967) align 4 %0, %struct.point967* noundef byval(%struct.point967) align 4 %1) #0 {
  %3 = bitcast %struct.point967* %0 to i8*
  %4 = bitcast %struct.point967* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo968(%struct.point968* noalias sret(%struct.point968) align 4 %0, %struct.point968* noundef byval(%struct.point968) align 4 %1) #0 {
  %3 = bitcast %struct.point968* %0 to i8*
  %4 = bitcast %struct.point968* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo969(%struct.point969* noalias sret(%struct.point969) align 4 %0, %struct.point969* noundef byval(%struct.point969) align 4 %1) #0 {
  %3 = bitcast %struct.point969* %0 to i8*
  %4 = bitcast %struct.point969* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo970(%struct.point970* noalias sret(%struct.point970) align 4 %0, %struct.point970* noundef byval(%struct.point970) align 4 %1) #0 {
  %3 = bitcast %struct.point970* %0 to i8*
  %4 = bitcast %struct.point970* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo971(%struct.point971* noalias sret(%struct.point971) align 4 %0, %struct.point971* noundef byval(%struct.point971) align 4 %1) #0 {
  %3 = bitcast %struct.point971* %0 to i8*
  %4 = bitcast %struct.point971* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo972(%struct.point972* noalias sret(%struct.point972) align 4 %0, %struct.point972* noundef byval(%struct.point972) align 4 %1) #0 {
  %3 = bitcast %struct.point972* %0 to i8*
  %4 = bitcast %struct.point972* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo973(%struct.point973* noalias sret(%struct.point973) align 4 %0, %struct.point973* noundef byval(%struct.point973) align 4 %1) #0 {
  %3 = bitcast %struct.point973* %0 to i8*
  %4 = bitcast %struct.point973* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo974(%struct.point974* noalias sret(%struct.point974) align 4 %0, %struct.point974* noundef byval(%struct.point974) align 4 %1) #0 {
  %3 = bitcast %struct.point974* %0 to i8*
  %4 = bitcast %struct.point974* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo975(%struct.point975* noalias sret(%struct.point975) align 4 %0, %struct.point975* noundef byval(%struct.point975) align 4 %1) #0 {
  %3 = bitcast %struct.point975* %0 to i8*
  %4 = bitcast %struct.point975* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo976(%struct.point976* noalias sret(%struct.point976) align 4 %0, %struct.point976* noundef byval(%struct.point976) align 4 %1) #0 {
  %3 = bitcast %struct.point976* %0 to i8*
  %4 = bitcast %struct.point976* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo977(%struct.point977* noalias sret(%struct.point977) align 4 %0, %struct.point977* noundef byval(%struct.point977) align 4 %1) #0 {
  %3 = bitcast %struct.point977* %0 to i8*
  %4 = bitcast %struct.point977* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo978(%struct.point978* noalias sret(%struct.point978) align 4 %0, %struct.point978* noundef byval(%struct.point978) align 4 %1) #0 {
  %3 = bitcast %struct.point978* %0 to i8*
  %4 = bitcast %struct.point978* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo979(%struct.point979* noalias sret(%struct.point979) align 4 %0, %struct.point979* noundef byval(%struct.point979) align 4 %1) #0 {
  %3 = bitcast %struct.point979* %0 to i8*
  %4 = bitcast %struct.point979* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo980(%struct.point980* noalias sret(%struct.point980) align 4 %0, %struct.point980* noundef byval(%struct.point980) align 4 %1) #0 {
  %3 = bitcast %struct.point980* %0 to i8*
  %4 = bitcast %struct.point980* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo981(%struct.point981* noalias sret(%struct.point981) align 4 %0, %struct.point981* noundef byval(%struct.point981) align 4 %1) #0 {
  %3 = bitcast %struct.point981* %0 to i8*
  %4 = bitcast %struct.point981* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo982(%struct.point982* noalias sret(%struct.point982) align 4 %0, %struct.point982* noundef byval(%struct.point982) align 4 %1) #0 {
  %3 = bitcast %struct.point982* %0 to i8*
  %4 = bitcast %struct.point982* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo983(%struct.point983* noalias sret(%struct.point983) align 4 %0, %struct.point983* noundef byval(%struct.point983) align 4 %1) #0 {
  %3 = bitcast %struct.point983* %0 to i8*
  %4 = bitcast %struct.point983* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo984(%struct.point984* noalias sret(%struct.point984) align 4 %0, %struct.point984* noundef byval(%struct.point984) align 4 %1) #0 {
  %3 = bitcast %struct.point984* %0 to i8*
  %4 = bitcast %struct.point984* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo985(%struct.point985* noalias sret(%struct.point985) align 4 %0, %struct.point985* noundef byval(%struct.point985) align 4 %1) #0 {
  %3 = bitcast %struct.point985* %0 to i8*
  %4 = bitcast %struct.point985* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo986(%struct.point986* noalias sret(%struct.point986) align 4 %0, %struct.point986* noundef byval(%struct.point986) align 4 %1) #0 {
  %3 = bitcast %struct.point986* %0 to i8*
  %4 = bitcast %struct.point986* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo987(%struct.point987* noalias sret(%struct.point987) align 4 %0, %struct.point987* noundef byval(%struct.point987) align 4 %1) #0 {
  %3 = bitcast %struct.point987* %0 to i8*
  %4 = bitcast %struct.point987* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo988(%struct.point988* noalias sret(%struct.point988) align 4 %0, %struct.point988* noundef byval(%struct.point988) align 4 %1) #0 {
  %3 = bitcast %struct.point988* %0 to i8*
  %4 = bitcast %struct.point988* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo989(%struct.point989* noalias sret(%struct.point989) align 4 %0, %struct.point989* noundef byval(%struct.point989) align 4 %1) #0 {
  %3 = bitcast %struct.point989* %0 to i8*
  %4 = bitcast %struct.point989* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo990(%struct.point990* noalias sret(%struct.point990) align 4 %0, %struct.point990* noundef byval(%struct.point990) align 4 %1) #0 {
  %3 = bitcast %struct.point990* %0 to i8*
  %4 = bitcast %struct.point990* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo991(%struct.point991* noalias sret(%struct.point991) align 4 %0, %struct.point991* noundef byval(%struct.point991) align 4 %1) #0 {
  %3 = bitcast %struct.point991* %0 to i8*
  %4 = bitcast %struct.point991* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo992(%struct.point992* noalias sret(%struct.point992) align 4 %0, %struct.point992* noundef byval(%struct.point992) align 4 %1) #0 {
  %3 = bitcast %struct.point992* %0 to i8*
  %4 = bitcast %struct.point992* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo993(%struct.point993* noalias sret(%struct.point993) align 4 %0, %struct.point993* noundef byval(%struct.point993) align 4 %1) #0 {
  %3 = bitcast %struct.point993* %0 to i8*
  %4 = bitcast %struct.point993* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo994(%struct.point994* noalias sret(%struct.point994) align 4 %0, %struct.point994* noundef byval(%struct.point994) align 4 %1) #0 {
  %3 = bitcast %struct.point994* %0 to i8*
  %4 = bitcast %struct.point994* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo995(%struct.point995* noalias sret(%struct.point995) align 4 %0, %struct.point995* noundef byval(%struct.point995) align 4 %1) #0 {
  %3 = bitcast %struct.point995* %0 to i8*
  %4 = bitcast %struct.point995* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo996(%struct.point996* noalias sret(%struct.point996) align 4 %0, %struct.point996* noundef byval(%struct.point996) align 4 %1) #0 {
  %3 = bitcast %struct.point996* %0 to i8*
  %4 = bitcast %struct.point996* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo997(%struct.point997* noalias sret(%struct.point997) align 4 %0, %struct.point997* noundef byval(%struct.point997) align 4 %1) #0 {
  %3 = bitcast %struct.point997* %0 to i8*
  %4 = bitcast %struct.point997* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo998(%struct.point998* noalias sret(%struct.point998) align 4 %0, %struct.point998* noundef byval(%struct.point998) align 4 %1) #0 {
  %3 = bitcast %struct.point998* %0 to i8*
  %4 = bitcast %struct.point998* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo999(%struct.point999* noalias sret(%struct.point999) align 4 %0, %struct.point999* noundef byval(%struct.point999) align 4 %1) #0 {
  %3 = bitcast %struct.point999* %0 to i8*
  %4 = bitcast %struct.point999* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1000(%struct.point1000* noalias sret(%struct.point1000) align 4 %0, %struct.point1000* noundef byval(%struct.point1000) align 4 %1) #0 {
  %3 = bitcast %struct.point1000* %0 to i8*
  %4 = bitcast %struct.point1000* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1001(%struct.point1001* noalias sret(%struct.point1001) align 4 %0, %struct.point1001* noundef byval(%struct.point1001) align 4 %1) #0 {
  %3 = bitcast %struct.point1001* %0 to i8*
  %4 = bitcast %struct.point1001* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1002(%struct.point1002* noalias sret(%struct.point1002) align 4 %0, %struct.point1002* noundef byval(%struct.point1002) align 4 %1) #0 {
  %3 = bitcast %struct.point1002* %0 to i8*
  %4 = bitcast %struct.point1002* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1003(%struct.point1003* noalias sret(%struct.point1003) align 4 %0, %struct.point1003* noundef byval(%struct.point1003) align 4 %1) #0 {
  %3 = bitcast %struct.point1003* %0 to i8*
  %4 = bitcast %struct.point1003* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1004(%struct.point1004* noalias sret(%struct.point1004) align 4 %0, %struct.point1004* noundef byval(%struct.point1004) align 4 %1) #0 {
  %3 = bitcast %struct.point1004* %0 to i8*
  %4 = bitcast %struct.point1004* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1005(%struct.point1005* noalias sret(%struct.point1005) align 4 %0, %struct.point1005* noundef byval(%struct.point1005) align 4 %1) #0 {
  %3 = bitcast %struct.point1005* %0 to i8*
  %4 = bitcast %struct.point1005* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1006(%struct.point1006* noalias sret(%struct.point1006) align 4 %0, %struct.point1006* noundef byval(%struct.point1006) align 4 %1) #0 {
  %3 = bitcast %struct.point1006* %0 to i8*
  %4 = bitcast %struct.point1006* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1007(%struct.point1007* noalias sret(%struct.point1007) align 4 %0, %struct.point1007* noundef byval(%struct.point1007) align 4 %1) #0 {
  %3 = bitcast %struct.point1007* %0 to i8*
  %4 = bitcast %struct.point1007* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1008(%struct.point1008* noalias sret(%struct.point1008) align 4 %0, %struct.point1008* noundef byval(%struct.point1008) align 4 %1) #0 {
  %3 = bitcast %struct.point1008* %0 to i8*
  %4 = bitcast %struct.point1008* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1009(%struct.point1009* noalias sret(%struct.point1009) align 4 %0, %struct.point1009* noundef byval(%struct.point1009) align 4 %1) #0 {
  %3 = bitcast %struct.point1009* %0 to i8*
  %4 = bitcast %struct.point1009* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1010(%struct.point1010* noalias sret(%struct.point1010) align 4 %0, %struct.point1010* noundef byval(%struct.point1010) align 4 %1) #0 {
  %3 = bitcast %struct.point1010* %0 to i8*
  %4 = bitcast %struct.point1010* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1011(%struct.point1011* noalias sret(%struct.point1011) align 4 %0, %struct.point1011* noundef byval(%struct.point1011) align 4 %1) #0 {
  %3 = bitcast %struct.point1011* %0 to i8*
  %4 = bitcast %struct.point1011* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1012(%struct.point1012* noalias sret(%struct.point1012) align 4 %0, %struct.point1012* noundef byval(%struct.point1012) align 4 %1) #0 {
  %3 = bitcast %struct.point1012* %0 to i8*
  %4 = bitcast %struct.point1012* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1013(%struct.point1013* noalias sret(%struct.point1013) align 4 %0, %struct.point1013* noundef byval(%struct.point1013) align 4 %1) #0 {
  %3 = bitcast %struct.point1013* %0 to i8*
  %4 = bitcast %struct.point1013* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1014(%struct.point1014* noalias sret(%struct.point1014) align 4 %0, %struct.point1014* noundef byval(%struct.point1014) align 4 %1) #0 {
  %3 = bitcast %struct.point1014* %0 to i8*
  %4 = bitcast %struct.point1014* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1015(%struct.point1015* noalias sret(%struct.point1015) align 4 %0, %struct.point1015* noundef byval(%struct.point1015) align 4 %1) #0 {
  %3 = bitcast %struct.point1015* %0 to i8*
  %4 = bitcast %struct.point1015* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1016(%struct.point1016* noalias sret(%struct.point1016) align 4 %0, %struct.point1016* noundef byval(%struct.point1016) align 4 %1) #0 {
  %3 = bitcast %struct.point1016* %0 to i8*
  %4 = bitcast %struct.point1016* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1017(%struct.point1017* noalias sret(%struct.point1017) align 4 %0, %struct.point1017* noundef byval(%struct.point1017) align 4 %1) #0 {
  %3 = bitcast %struct.point1017* %0 to i8*
  %4 = bitcast %struct.point1017* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1018(%struct.point1018* noalias sret(%struct.point1018) align 4 %0, %struct.point1018* noundef byval(%struct.point1018) align 4 %1) #0 {
  %3 = bitcast %struct.point1018* %0 to i8*
  %4 = bitcast %struct.point1018* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1019(%struct.point1019* noalias sret(%struct.point1019) align 4 %0, %struct.point1019* noundef byval(%struct.point1019) align 4 %1) #0 {
  %3 = bitcast %struct.point1019* %0 to i8*
  %4 = bitcast %struct.point1019* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1020(%struct.point1020* noalias sret(%struct.point1020) align 4 %0, %struct.point1020* noundef byval(%struct.point1020) align 4 %1) #0 {
  %3 = bitcast %struct.point1020* %0 to i8*
  %4 = bitcast %struct.point1020* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1021(%struct.point1021* noalias sret(%struct.point1021) align 4 %0, %struct.point1021* noundef byval(%struct.point1021) align 4 %1) #0 {
  %3 = bitcast %struct.point1021* %0 to i8*
  %4 = bitcast %struct.point1021* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1022(%struct.point1022* noalias sret(%struct.point1022) align 4 %0, %struct.point1022* noundef byval(%struct.point1022) align 4 %1) #0 {
  %3 = bitcast %struct.point1022* %0 to i8*
  %4 = bitcast %struct.point1022* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1023(%struct.point1023* noalias sret(%struct.point1023) align 4 %0, %struct.point1023* noundef byval(%struct.point1023) align 4 %1) #0 {
  %3 = bitcast %struct.point1023* %0 to i8*
  %4 = bitcast %struct.point1023* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1024(%struct.point1024* noalias sret(%struct.point1024) align 4 %0, %struct.point1024* noundef byval(%struct.point1024) align 4 %1) #0 {
  %3 = bitcast %struct.point1024* %0 to i8*
  %4 = bitcast %struct.point1024* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1025(%struct.point1025* noalias sret(%struct.point1025) align 4 %0, %struct.point1025* noundef byval(%struct.point1025) align 4 %1) #0 {
  %3 = bitcast %struct.point1025* %0 to i8*
  %4 = bitcast %struct.point1025* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1026(%struct.point1026* noalias sret(%struct.point1026) align 4 %0, %struct.point1026* noundef byval(%struct.point1026) align 4 %1) #0 {
  %3 = bitcast %struct.point1026* %0 to i8*
  %4 = bitcast %struct.point1026* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1027(%struct.point1027* noalias sret(%struct.point1027) align 4 %0, %struct.point1027* noundef byval(%struct.point1027) align 4 %1) #0 {
  %3 = bitcast %struct.point1027* %0 to i8*
  %4 = bitcast %struct.point1027* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1028(%struct.point1028* noalias sret(%struct.point1028) align 4 %0, %struct.point1028* noundef byval(%struct.point1028) align 4 %1) #0 {
  %3 = bitcast %struct.point1028* %0 to i8*
  %4 = bitcast %struct.point1028* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1029(%struct.point1029* noalias sret(%struct.point1029) align 4 %0, %struct.point1029* noundef byval(%struct.point1029) align 4 %1) #0 {
  %3 = bitcast %struct.point1029* %0 to i8*
  %4 = bitcast %struct.point1029* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1030(%struct.point1030* noalias sret(%struct.point1030) align 4 %0, %struct.point1030* noundef byval(%struct.point1030) align 4 %1) #0 {
  %3 = bitcast %struct.point1030* %0 to i8*
  %4 = bitcast %struct.point1030* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 12, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1031(%struct.point1031* noalias sret(%struct.point1031) align 4 %0, %struct.point1031* noundef byval(%struct.point1031) align 4 %1) #0 {
  %3 = bitcast %struct.point1031* %0 to i8*
  %4 = bitcast %struct.point1031* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1032(%struct.point1032* noalias sret(%struct.point1032) align 4 %0, %struct.point1032* noundef byval(%struct.point1032) align 4 %1) #0 {
  %3 = bitcast %struct.point1032* %0 to i8*
  %4 = bitcast %struct.point1032* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1033(%struct.point1033* noalias sret(%struct.point1033) align 4 %0, %struct.point1033* noundef byval(%struct.point1033) align 4 %1) #0 {
  %3 = bitcast %struct.point1033* %0 to i8*
  %4 = bitcast %struct.point1033* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1034(%struct.point1034* noalias sret(%struct.point1034) align 4 %0, %struct.point1034* noundef byval(%struct.point1034) align 4 %1) #0 {
  %3 = bitcast %struct.point1034* %0 to i8*
  %4 = bitcast %struct.point1034* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1035(%struct.point1035* noalias sret(%struct.point1035) align 4 %0, %struct.point1035* noundef byval(%struct.point1035) align 4 %1) #0 {
  %3 = bitcast %struct.point1035* %0 to i8*
  %4 = bitcast %struct.point1035* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1036(%struct.point1036* noalias sret(%struct.point1036) align 4 %0, %struct.point1036* noundef byval(%struct.point1036) align 4 %1) #0 {
  %3 = bitcast %struct.point1036* %0 to i8*
  %4 = bitcast %struct.point1036* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1037(%struct.point1037* noalias sret(%struct.point1037) align 4 %0, %struct.point1037* noundef byval(%struct.point1037) align 4 %1) #0 {
  %3 = bitcast %struct.point1037* %0 to i8*
  %4 = bitcast %struct.point1037* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1038(%struct.point1038* noalias sret(%struct.point1038) align 4 %0, %struct.point1038* noundef byval(%struct.point1038) align 4 %1) #0 {
  %3 = bitcast %struct.point1038* %0 to i8*
  %4 = bitcast %struct.point1038* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1039(%struct.point1039* noalias sret(%struct.point1039) align 4 %0, %struct.point1039* noundef byval(%struct.point1039) align 4 %1) #0 {
  %3 = bitcast %struct.point1039* %0 to i8*
  %4 = bitcast %struct.point1039* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1040(%struct.point1040* noalias sret(%struct.point1040) align 4 %0, %struct.point1040* noundef byval(%struct.point1040) align 4 %1) #0 {
  %3 = bitcast %struct.point1040* %0 to i8*
  %4 = bitcast %struct.point1040* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1041(%struct.point1041* noalias sret(%struct.point1041) align 4 %0, %struct.point1041* noundef byval(%struct.point1041) align 4 %1) #0 {
  %3 = bitcast %struct.point1041* %0 to i8*
  %4 = bitcast %struct.point1041* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1042(%struct.point1042* noalias sret(%struct.point1042) align 4 %0, %struct.point1042* noundef byval(%struct.point1042) align 4 %1) #0 {
  %3 = bitcast %struct.point1042* %0 to i8*
  %4 = bitcast %struct.point1042* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1043(%struct.point1043* noalias sret(%struct.point1043) align 4 %0, %struct.point1043* noundef byval(%struct.point1043) align 4 %1) #0 {
  %3 = bitcast %struct.point1043* %0 to i8*
  %4 = bitcast %struct.point1043* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1044(%struct.point1044* noalias sret(%struct.point1044) align 4 %0, %struct.point1044* noundef byval(%struct.point1044) align 4 %1) #0 {
  %3 = bitcast %struct.point1044* %0 to i8*
  %4 = bitcast %struct.point1044* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1045(%struct.point1045* noalias sret(%struct.point1045) align 4 %0, %struct.point1045* noundef byval(%struct.point1045) align 4 %1) #0 {
  %3 = bitcast %struct.point1045* %0 to i8*
  %4 = bitcast %struct.point1045* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1046(%struct.point1046* noalias sret(%struct.point1046) align 4 %0, %struct.point1046* noundef byval(%struct.point1046) align 4 %1) #0 {
  %3 = bitcast %struct.point1046* %0 to i8*
  %4 = bitcast %struct.point1046* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 16, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1047(%struct.point1047* noalias sret(%struct.point1047) align 4 %0, %struct.point1047* noundef byval(%struct.point1047) align 4 %1) #0 {
  %3 = bitcast %struct.point1047* %0 to i8*
  %4 = bitcast %struct.point1047* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1048(%struct.point1048* noalias sret(%struct.point1048) align 4 %0, %struct.point1048* noundef byval(%struct.point1048) align 4 %1) #0 {
  %3 = bitcast %struct.point1048* %0 to i8*
  %4 = bitcast %struct.point1048* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1049(%struct.point1049* noalias sret(%struct.point1049) align 4 %0, %struct.point1049* noundef byval(%struct.point1049) align 4 %1) #0 {
  %3 = bitcast %struct.point1049* %0 to i8*
  %4 = bitcast %struct.point1049* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define hidden void @demo1050(%struct.point1050* noalias sret(%struct.point1050) align 4 %0, %struct.point1050* noundef byval(%struct.point1050) align 4 %1) #0 {
  %3 = bitcast %struct.point1050* %0 to i8*
  %4 = bitcast %struct.point1050* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i32(i8* align 4 %3, i8* align 4 %4, i32 20, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="none" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-cpu"="generic" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0}
!llvm.ident = !{!1}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
