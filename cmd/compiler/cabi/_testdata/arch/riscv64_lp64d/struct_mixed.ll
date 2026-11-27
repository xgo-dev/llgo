; ModuleID = '../../wrap/struct_mixed.c'
source_filename = "../../wrap/struct_mixed.c"
target datalayout = "e-m:e-p:64:64-i64:64-i128:128-n64-S128"
target triple = "riscv64-unknown-unknown-elf"

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
define dso_local i64 @demo1(i64 %0) #0 {
  %2 = alloca %struct.point1, align 2
  %3 = alloca %struct.point1, align 2
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point1* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 4, i1 false)
  %8 = bitcast %struct.point1* %2 to i8*
  %9 = bitcast %struct.point1* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 4, i1 false)
  %10 = bitcast i64* %5 to i8*
  %11 = bitcast %struct.point1* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 4, i1 false)
  %12 = load i64, i64* %5, align 8
  ret i64 %12
}

; Function Attrs: argmemonly nofree nounwind willreturn
declare void @llvm.memcpy.p0i8.p0i8.i64(i8* noalias nocapture writeonly, i8* noalias nocapture readonly, i64, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo2(i64 %0) #0 {
  %2 = alloca %struct.point2, align 4
  %3 = alloca %struct.point2, align 4
  %4 = bitcast %struct.point2* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point2* %2 to i8*
  %6 = bitcast %struct.point2* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point2* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo3([2 x i64] %0) #0 {
  %2 = alloca %struct.point3, align 8
  %3 = alloca %struct.point3, align 8
  %4 = bitcast %struct.point3* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point3* %2 to i8*
  %6 = bitcast %struct.point3* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point3* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local { i8, float } @demo4(i8 %0, float %1) #0 {
  %3 = alloca %struct.point4, align 4
  %4 = alloca %struct.point4, align 4
  %5 = bitcast %struct.point4* %4 to { i8, float }*
  %6 = getelementptr inbounds { i8, float }, { i8, float }* %5, i32 0, i32 0
  store i8 %0, i8* %6, align 4
  %7 = getelementptr inbounds { i8, float }, { i8, float }* %5, i32 0, i32 1
  store float %1, float* %7, align 4
  %8 = bitcast %struct.point4* %3 to i8*
  %9 = bitcast %struct.point4* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 8, i1 false)
  %10 = bitcast %struct.point4* %3 to { i8, float }*
  %11 = getelementptr inbounds { i8, float }, { i8, float }* %10, i32 0, i32 0
  %12 = load i8, i8* %11, align 4
  %13 = getelementptr inbounds { i8, float }, { i8, float }* %10, i32 0, i32 1
  %14 = load float, float* %13, align 4
  %15 = insertvalue { i8, float } undef, i8 %12, 0
  %16 = insertvalue { i8, float } %15, float %14, 1
  ret { i8, float } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { i8, double } @demo5(i8 %0, double %1) #0 {
  %3 = alloca %struct.point5, align 8
  %4 = alloca %struct.point5, align 8
  %5 = bitcast %struct.point5* %4 to { i8, double }*
  %6 = getelementptr inbounds { i8, double }, { i8, double }* %5, i32 0, i32 0
  store i8 %0, i8* %6, align 8
  %7 = getelementptr inbounds { i8, double }, { i8, double }* %5, i32 0, i32 1
  store double %1, double* %7, align 8
  %8 = bitcast %struct.point5* %3 to i8*
  %9 = bitcast %struct.point5* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point5* %3 to { i8, double }*
  %11 = getelementptr inbounds { i8, double }, { i8, double }* %10, i32 0, i32 0
  %12 = load i8, i8* %11, align 8
  %13 = getelementptr inbounds { i8, double }, { i8, double }* %10, i32 0, i32 1
  %14 = load double, double* %13, align 8
  %15 = insertvalue { i8, double } undef, i8 %12, 0
  %16 = insertvalue { i8, double } %15, double %14, 1
  ret { i8, double } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo6([2 x i64] %0) #0 {
  %2 = alloca %struct.point6, align 8
  %3 = alloca %struct.point6, align 8
  %4 = bitcast %struct.point6* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point6* %2 to i8*
  %6 = bitcast %struct.point6* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point6* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo7(i64 %0) #0 {
  %2 = alloca %struct.point7, align 2
  %3 = alloca %struct.point7, align 2
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point7* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 4, i1 false)
  %8 = bitcast %struct.point7* %2 to i8*
  %9 = bitcast %struct.point7* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 4, i1 false)
  %10 = bitcast i64* %5 to i8*
  %11 = bitcast %struct.point7* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 4, i1 false)
  %12 = load i64, i64* %5, align 8
  ret i64 %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo8(i64 %0) #0 {
  %2 = alloca %struct.point8, align 4
  %3 = alloca %struct.point8, align 4
  %4 = bitcast %struct.point8* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point8* %2 to i8*
  %6 = bitcast %struct.point8* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point8* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo9([2 x i64] %0) #0 {
  %2 = alloca %struct.point9, align 8
  %3 = alloca %struct.point9, align 8
  %4 = bitcast %struct.point9* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point9* %2 to i8*
  %6 = bitcast %struct.point9* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point9* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local { i16, float } @demo10(i16 %0, float %1) #0 {
  %3 = alloca %struct.point10, align 4
  %4 = alloca %struct.point10, align 4
  %5 = bitcast %struct.point10* %4 to { i16, float }*
  %6 = getelementptr inbounds { i16, float }, { i16, float }* %5, i32 0, i32 0
  store i16 %0, i16* %6, align 4
  %7 = getelementptr inbounds { i16, float }, { i16, float }* %5, i32 0, i32 1
  store float %1, float* %7, align 4
  %8 = bitcast %struct.point10* %3 to i8*
  %9 = bitcast %struct.point10* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 8, i1 false)
  %10 = bitcast %struct.point10* %3 to { i16, float }*
  %11 = getelementptr inbounds { i16, float }, { i16, float }* %10, i32 0, i32 0
  %12 = load i16, i16* %11, align 4
  %13 = getelementptr inbounds { i16, float }, { i16, float }* %10, i32 0, i32 1
  %14 = load float, float* %13, align 4
  %15 = insertvalue { i16, float } undef, i16 %12, 0
  %16 = insertvalue { i16, float } %15, float %14, 1
  ret { i16, float } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { i16, double } @demo11(i16 %0, double %1) #0 {
  %3 = alloca %struct.point11, align 8
  %4 = alloca %struct.point11, align 8
  %5 = bitcast %struct.point11* %4 to { i16, double }*
  %6 = getelementptr inbounds { i16, double }, { i16, double }* %5, i32 0, i32 0
  store i16 %0, i16* %6, align 8
  %7 = getelementptr inbounds { i16, double }, { i16, double }* %5, i32 0, i32 1
  store double %1, double* %7, align 8
  %8 = bitcast %struct.point11* %3 to i8*
  %9 = bitcast %struct.point11* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point11* %3 to { i16, double }*
  %11 = getelementptr inbounds { i16, double }, { i16, double }* %10, i32 0, i32 0
  %12 = load i16, i16* %11, align 8
  %13 = getelementptr inbounds { i16, double }, { i16, double }* %10, i32 0, i32 1
  %14 = load double, double* %13, align 8
  %15 = insertvalue { i16, double } undef, i16 %12, 0
  %16 = insertvalue { i16, double } %15, double %14, 1
  ret { i16, double } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo12([2 x i64] %0) #0 {
  %2 = alloca %struct.point12, align 8
  %3 = alloca %struct.point12, align 8
  %4 = bitcast %struct.point12* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point12* %2 to i8*
  %6 = bitcast %struct.point12* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point12* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo13(i64 %0) #0 {
  %2 = alloca %struct.point13, align 4
  %3 = alloca %struct.point13, align 4
  %4 = bitcast %struct.point13* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point13* %2 to i8*
  %6 = bitcast %struct.point13* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point13* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo14(i64 %0) #0 {
  %2 = alloca %struct.point14, align 4
  %3 = alloca %struct.point14, align 4
  %4 = bitcast %struct.point14* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point14* %2 to i8*
  %6 = bitcast %struct.point14* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point14* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo15([2 x i64] %0) #0 {
  %2 = alloca %struct.point15, align 8
  %3 = alloca %struct.point15, align 8
  %4 = bitcast %struct.point15* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point15* %2 to i8*
  %6 = bitcast %struct.point15* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point15* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local { i32, float } @demo16(i32 %0, float %1) #0 {
  %3 = alloca %struct.point16, align 4
  %4 = alloca %struct.point16, align 4
  %5 = bitcast %struct.point16* %4 to { i32, float }*
  %6 = getelementptr inbounds { i32, float }, { i32, float }* %5, i32 0, i32 0
  store i32 %0, i32* %6, align 4
  %7 = getelementptr inbounds { i32, float }, { i32, float }* %5, i32 0, i32 1
  store float %1, float* %7, align 4
  %8 = bitcast %struct.point16* %3 to i8*
  %9 = bitcast %struct.point16* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 8, i1 false)
  %10 = bitcast %struct.point16* %3 to { i32, float }*
  %11 = getelementptr inbounds { i32, float }, { i32, float }* %10, i32 0, i32 0
  %12 = load i32, i32* %11, align 4
  %13 = getelementptr inbounds { i32, float }, { i32, float }* %10, i32 0, i32 1
  %14 = load float, float* %13, align 4
  %15 = insertvalue { i32, float } undef, i32 %12, 0
  %16 = insertvalue { i32, float } %15, float %14, 1
  ret { i32, float } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { i32, double } @demo17(i32 %0, double %1) #0 {
  %3 = alloca %struct.point17, align 8
  %4 = alloca %struct.point17, align 8
  %5 = bitcast %struct.point17* %4 to { i32, double }*
  %6 = getelementptr inbounds { i32, double }, { i32, double }* %5, i32 0, i32 0
  store i32 %0, i32* %6, align 8
  %7 = getelementptr inbounds { i32, double }, { i32, double }* %5, i32 0, i32 1
  store double %1, double* %7, align 8
  %8 = bitcast %struct.point17* %3 to i8*
  %9 = bitcast %struct.point17* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point17* %3 to { i32, double }*
  %11 = getelementptr inbounds { i32, double }, { i32, double }* %10, i32 0, i32 0
  %12 = load i32, i32* %11, align 8
  %13 = getelementptr inbounds { i32, double }, { i32, double }* %10, i32 0, i32 1
  %14 = load double, double* %13, align 8
  %15 = insertvalue { i32, double } undef, i32 %12, 0
  %16 = insertvalue { i32, double } %15, double %14, 1
  ret { i32, double } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo18([2 x i64] %0) #0 {
  %2 = alloca %struct.point18, align 8
  %3 = alloca %struct.point18, align 8
  %4 = bitcast %struct.point18* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point18* %2 to i8*
  %6 = bitcast %struct.point18* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point18* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo19([2 x i64] %0) #0 {
  %2 = alloca %struct.point19, align 8
  %3 = alloca %struct.point19, align 8
  %4 = bitcast %struct.point19* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point19* %2 to i8*
  %6 = bitcast %struct.point19* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point19* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo20([2 x i64] %0) #0 {
  %2 = alloca %struct.point20, align 8
  %3 = alloca %struct.point20, align 8
  %4 = bitcast %struct.point20* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point20* %2 to i8*
  %6 = bitcast %struct.point20* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point20* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo21([2 x i64] %0) #0 {
  %2 = alloca %struct.point21, align 8
  %3 = alloca %struct.point21, align 8
  %4 = bitcast %struct.point21* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point21* %2 to i8*
  %6 = bitcast %struct.point21* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point21* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local { i64, float } @demo22(i64 %0, float %1) #0 {
  %3 = alloca %struct.point22, align 8
  %4 = alloca %struct.point22, align 8
  %5 = bitcast %struct.point22* %4 to { i64, float }*
  %6 = getelementptr inbounds { i64, float }, { i64, float }* %5, i32 0, i32 0
  store i64 %0, i64* %6, align 8
  %7 = getelementptr inbounds { i64, float }, { i64, float }* %5, i32 0, i32 1
  store float %1, float* %7, align 8
  %8 = bitcast %struct.point22* %3 to i8*
  %9 = bitcast %struct.point22* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point22* %3 to { i64, float }*
  %11 = getelementptr inbounds { i64, float }, { i64, float }* %10, i32 0, i32 0
  %12 = load i64, i64* %11, align 8
  %13 = getelementptr inbounds { i64, float }, { i64, float }* %10, i32 0, i32 1
  %14 = load float, float* %13, align 8
  %15 = insertvalue { i64, float } undef, i64 %12, 0
  %16 = insertvalue { i64, float } %15, float %14, 1
  ret { i64, float } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { i64, double } @demo23(i64 %0, double %1) #0 {
  %3 = alloca %struct.point23, align 8
  %4 = alloca %struct.point23, align 8
  %5 = bitcast %struct.point23* %4 to { i64, double }*
  %6 = getelementptr inbounds { i64, double }, { i64, double }* %5, i32 0, i32 0
  store i64 %0, i64* %6, align 8
  %7 = getelementptr inbounds { i64, double }, { i64, double }* %5, i32 0, i32 1
  store double %1, double* %7, align 8
  %8 = bitcast %struct.point23* %3 to i8*
  %9 = bitcast %struct.point23* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point23* %3 to { i64, double }*
  %11 = getelementptr inbounds { i64, double }, { i64, double }* %10, i32 0, i32 0
  %12 = load i64, i64* %11, align 8
  %13 = getelementptr inbounds { i64, double }, { i64, double }* %10, i32 0, i32 1
  %14 = load double, double* %13, align 8
  %15 = insertvalue { i64, double } undef, i64 %12, 0
  %16 = insertvalue { i64, double } %15, double %14, 1
  ret { i64, double } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo24([2 x i64] %0) #0 {
  %2 = alloca %struct.point24, align 8
  %3 = alloca %struct.point24, align 8
  %4 = bitcast %struct.point24* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point24* %2 to i8*
  %6 = bitcast %struct.point24* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point24* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local { float, i8 } @demo25(float %0, i8 %1) #0 {
  %3 = alloca %struct.point25, align 4
  %4 = alloca %struct.point25, align 4
  %5 = bitcast %struct.point25* %4 to { float, i8 }*
  %6 = getelementptr inbounds { float, i8 }, { float, i8 }* %5, i32 0, i32 0
  store float %0, float* %6, align 4
  %7 = getelementptr inbounds { float, i8 }, { float, i8 }* %5, i32 0, i32 1
  store i8 %1, i8* %7, align 4
  %8 = bitcast %struct.point25* %3 to i8*
  %9 = bitcast %struct.point25* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 8, i1 false)
  %10 = bitcast %struct.point25* %3 to { float, i8 }*
  %11 = getelementptr inbounds { float, i8 }, { float, i8 }* %10, i32 0, i32 0
  %12 = load float, float* %11, align 4
  %13 = getelementptr inbounds { float, i8 }, { float, i8 }* %10, i32 0, i32 1
  %14 = load i8, i8* %13, align 4
  %15 = insertvalue { float, i8 } undef, float %12, 0
  %16 = insertvalue { float, i8 } %15, i8 %14, 1
  ret { float, i8 } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { float, i16 } @demo26(float %0, i16 %1) #0 {
  %3 = alloca %struct.point26, align 4
  %4 = alloca %struct.point26, align 4
  %5 = bitcast %struct.point26* %4 to { float, i16 }*
  %6 = getelementptr inbounds { float, i16 }, { float, i16 }* %5, i32 0, i32 0
  store float %0, float* %6, align 4
  %7 = getelementptr inbounds { float, i16 }, { float, i16 }* %5, i32 0, i32 1
  store i16 %1, i16* %7, align 4
  %8 = bitcast %struct.point26* %3 to i8*
  %9 = bitcast %struct.point26* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 8, i1 false)
  %10 = bitcast %struct.point26* %3 to { float, i16 }*
  %11 = getelementptr inbounds { float, i16 }, { float, i16 }* %10, i32 0, i32 0
  %12 = load float, float* %11, align 4
  %13 = getelementptr inbounds { float, i16 }, { float, i16 }* %10, i32 0, i32 1
  %14 = load i16, i16* %13, align 4
  %15 = insertvalue { float, i16 } undef, float %12, 0
  %16 = insertvalue { float, i16 } %15, i16 %14, 1
  ret { float, i16 } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { float, i32 } @demo27(float %0, i32 %1) #0 {
  %3 = alloca %struct.point27, align 4
  %4 = alloca %struct.point27, align 4
  %5 = bitcast %struct.point27* %4 to { float, i32 }*
  %6 = getelementptr inbounds { float, i32 }, { float, i32 }* %5, i32 0, i32 0
  store float %0, float* %6, align 4
  %7 = getelementptr inbounds { float, i32 }, { float, i32 }* %5, i32 0, i32 1
  store i32 %1, i32* %7, align 4
  %8 = bitcast %struct.point27* %3 to i8*
  %9 = bitcast %struct.point27* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 8, i1 false)
  %10 = bitcast %struct.point27* %3 to { float, i32 }*
  %11 = getelementptr inbounds { float, i32 }, { float, i32 }* %10, i32 0, i32 0
  %12 = load float, float* %11, align 4
  %13 = getelementptr inbounds { float, i32 }, { float, i32 }* %10, i32 0, i32 1
  %14 = load i32, i32* %13, align 4
  %15 = insertvalue { float, i32 } undef, float %12, 0
  %16 = insertvalue { float, i32 } %15, i32 %14, 1
  ret { float, i32 } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { float, i64 } @demo28(float %0, i64 %1) #0 {
  %3 = alloca %struct.point28, align 8
  %4 = alloca %struct.point28, align 8
  %5 = bitcast %struct.point28* %4 to { float, i64 }*
  %6 = getelementptr inbounds { float, i64 }, { float, i64 }* %5, i32 0, i32 0
  store float %0, float* %6, align 8
  %7 = getelementptr inbounds { float, i64 }, { float, i64 }* %5, i32 0, i32 1
  store i64 %1, i64* %7, align 8
  %8 = bitcast %struct.point28* %3 to i8*
  %9 = bitcast %struct.point28* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point28* %3 to { float, i64 }*
  %11 = getelementptr inbounds { float, i64 }, { float, i64 }* %10, i32 0, i32 0
  %12 = load float, float* %11, align 8
  %13 = getelementptr inbounds { float, i64 }, { float, i64 }* %10, i32 0, i32 1
  %14 = load i64, i64* %13, align 8
  %15 = insertvalue { float, i64 } undef, float %12, 0
  %16 = insertvalue { float, i64 } %15, i64 %14, 1
  ret { float, i64 } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { float, double } @demo29(float %0, double %1) #0 {
  %3 = alloca %struct.point29, align 8
  %4 = alloca %struct.point29, align 8
  %5 = bitcast %struct.point29* %4 to { float, double }*
  %6 = getelementptr inbounds { float, double }, { float, double }* %5, i32 0, i32 0
  store float %0, float* %6, align 8
  %7 = getelementptr inbounds { float, double }, { float, double }* %5, i32 0, i32 1
  store double %1, double* %7, align 8
  %8 = bitcast %struct.point29* %3 to i8*
  %9 = bitcast %struct.point29* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point29* %3 to { float, double }*
  %11 = getelementptr inbounds { float, double }, { float, double }* %10, i32 0, i32 0
  %12 = load float, float* %11, align 8
  %13 = getelementptr inbounds { float, double }, { float, double }* %10, i32 0, i32 1
  %14 = load double, double* %13, align 8
  %15 = insertvalue { float, double } undef, float %12, 0
  %16 = insertvalue { float, double } %15, double %14, 1
  ret { float, double } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo30([2 x i64] %0) #0 {
  %2 = alloca %struct.point30, align 8
  %3 = alloca %struct.point30, align 8
  %4 = bitcast %struct.point30* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point30* %2 to i8*
  %6 = bitcast %struct.point30* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point30* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local { double, i8 } @demo31(double %0, i8 %1) #0 {
  %3 = alloca %struct.point31, align 8
  %4 = alloca %struct.point31, align 8
  %5 = bitcast %struct.point31* %4 to { double, i8 }*
  %6 = getelementptr inbounds { double, i8 }, { double, i8 }* %5, i32 0, i32 0
  store double %0, double* %6, align 8
  %7 = getelementptr inbounds { double, i8 }, { double, i8 }* %5, i32 0, i32 1
  store i8 %1, i8* %7, align 8
  %8 = bitcast %struct.point31* %3 to i8*
  %9 = bitcast %struct.point31* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point31* %3 to { double, i8 }*
  %11 = getelementptr inbounds { double, i8 }, { double, i8 }* %10, i32 0, i32 0
  %12 = load double, double* %11, align 8
  %13 = getelementptr inbounds { double, i8 }, { double, i8 }* %10, i32 0, i32 1
  %14 = load i8, i8* %13, align 8
  %15 = insertvalue { double, i8 } undef, double %12, 0
  %16 = insertvalue { double, i8 } %15, i8 %14, 1
  ret { double, i8 } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { double, i16 } @demo32(double %0, i16 %1) #0 {
  %3 = alloca %struct.point32, align 8
  %4 = alloca %struct.point32, align 8
  %5 = bitcast %struct.point32* %4 to { double, i16 }*
  %6 = getelementptr inbounds { double, i16 }, { double, i16 }* %5, i32 0, i32 0
  store double %0, double* %6, align 8
  %7 = getelementptr inbounds { double, i16 }, { double, i16 }* %5, i32 0, i32 1
  store i16 %1, i16* %7, align 8
  %8 = bitcast %struct.point32* %3 to i8*
  %9 = bitcast %struct.point32* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point32* %3 to { double, i16 }*
  %11 = getelementptr inbounds { double, i16 }, { double, i16 }* %10, i32 0, i32 0
  %12 = load double, double* %11, align 8
  %13 = getelementptr inbounds { double, i16 }, { double, i16 }* %10, i32 0, i32 1
  %14 = load i16, i16* %13, align 8
  %15 = insertvalue { double, i16 } undef, double %12, 0
  %16 = insertvalue { double, i16 } %15, i16 %14, 1
  ret { double, i16 } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { double, i32 } @demo33(double %0, i32 %1) #0 {
  %3 = alloca %struct.point33, align 8
  %4 = alloca %struct.point33, align 8
  %5 = bitcast %struct.point33* %4 to { double, i32 }*
  %6 = getelementptr inbounds { double, i32 }, { double, i32 }* %5, i32 0, i32 0
  store double %0, double* %6, align 8
  %7 = getelementptr inbounds { double, i32 }, { double, i32 }* %5, i32 0, i32 1
  store i32 %1, i32* %7, align 8
  %8 = bitcast %struct.point33* %3 to i8*
  %9 = bitcast %struct.point33* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point33* %3 to { double, i32 }*
  %11 = getelementptr inbounds { double, i32 }, { double, i32 }* %10, i32 0, i32 0
  %12 = load double, double* %11, align 8
  %13 = getelementptr inbounds { double, i32 }, { double, i32 }* %10, i32 0, i32 1
  %14 = load i32, i32* %13, align 8
  %15 = insertvalue { double, i32 } undef, double %12, 0
  %16 = insertvalue { double, i32 } %15, i32 %14, 1
  ret { double, i32 } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { double, i64 } @demo34(double %0, i64 %1) #0 {
  %3 = alloca %struct.point34, align 8
  %4 = alloca %struct.point34, align 8
  %5 = bitcast %struct.point34* %4 to { double, i64 }*
  %6 = getelementptr inbounds { double, i64 }, { double, i64 }* %5, i32 0, i32 0
  store double %0, double* %6, align 8
  %7 = getelementptr inbounds { double, i64 }, { double, i64 }* %5, i32 0, i32 1
  store i64 %1, i64* %7, align 8
  %8 = bitcast %struct.point34* %3 to i8*
  %9 = bitcast %struct.point34* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point34* %3 to { double, i64 }*
  %11 = getelementptr inbounds { double, i64 }, { double, i64 }* %10, i32 0, i32 0
  %12 = load double, double* %11, align 8
  %13 = getelementptr inbounds { double, i64 }, { double, i64 }* %10, i32 0, i32 1
  %14 = load i64, i64* %13, align 8
  %15 = insertvalue { double, i64 } undef, double %12, 0
  %16 = insertvalue { double, i64 } %15, i64 %14, 1
  ret { double, i64 } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local { double, float } @demo35(double %0, float %1) #0 {
  %3 = alloca %struct.point35, align 8
  %4 = alloca %struct.point35, align 8
  %5 = bitcast %struct.point35* %4 to { double, float }*
  %6 = getelementptr inbounds { double, float }, { double, float }* %5, i32 0, i32 0
  store double %0, double* %6, align 8
  %7 = getelementptr inbounds { double, float }, { double, float }* %5, i32 0, i32 1
  store float %1, float* %7, align 8
  %8 = bitcast %struct.point35* %3 to i8*
  %9 = bitcast %struct.point35* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %8, i8* align 8 %9, i64 16, i1 false)
  %10 = bitcast %struct.point35* %3 to { double, float }*
  %11 = getelementptr inbounds { double, float }, { double, float }* %10, i32 0, i32 0
  %12 = load double, double* %11, align 8
  %13 = getelementptr inbounds { double, float }, { double, float }* %10, i32 0, i32 1
  %14 = load float, float* %13, align 8
  %15 = insertvalue { double, float } undef, double %12, 0
  %16 = insertvalue { double, float } %15, float %14, 1
  ret { double, float } %16
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo36([2 x i64] %0) #0 {
  %2 = alloca %struct.point36, align 8
  %3 = alloca %struct.point36, align 8
  %4 = bitcast %struct.point36* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point36* %2 to i8*
  %6 = bitcast %struct.point36* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point36* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo37([2 x i64] %0) #0 {
  %2 = alloca %struct.point37, align 8
  %3 = alloca %struct.point37, align 8
  %4 = bitcast %struct.point37* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point37* %2 to i8*
  %6 = bitcast %struct.point37* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point37* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo38([2 x i64] %0) #0 {
  %2 = alloca %struct.point38, align 8
  %3 = alloca %struct.point38, align 8
  %4 = bitcast %struct.point38* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point38* %2 to i8*
  %6 = bitcast %struct.point38* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point38* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo39([2 x i64] %0) #0 {
  %2 = alloca %struct.point39, align 8
  %3 = alloca %struct.point39, align 8
  %4 = bitcast %struct.point39* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point39* %2 to i8*
  %6 = bitcast %struct.point39* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point39* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo40([2 x i64] %0) #0 {
  %2 = alloca %struct.point40, align 8
  %3 = alloca %struct.point40, align 8
  %4 = bitcast %struct.point40* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point40* %2 to i8*
  %6 = bitcast %struct.point40* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point40* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo41([2 x i64] %0) #0 {
  %2 = alloca %struct.point41, align 8
  %3 = alloca %struct.point41, align 8
  %4 = bitcast %struct.point41* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point41* %2 to i8*
  %6 = bitcast %struct.point41* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point41* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo42([2 x i64] %0) #0 {
  %2 = alloca %struct.point42, align 8
  %3 = alloca %struct.point42, align 8
  %4 = bitcast %struct.point42* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %5 = bitcast %struct.point42* %2 to i8*
  %6 = bitcast %struct.point42* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %5, i8* align 8 %6, i64 16, i1 false)
  %7 = bitcast %struct.point42* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 8
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo43(i64 %0) #0 {
  %2 = alloca %struct.point43, align 2
  %3 = alloca %struct.point43, align 2
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point43* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 6, i1 false)
  %8 = bitcast %struct.point43* %2 to i8*
  %9 = bitcast %struct.point43* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 6, i1 false)
  %10 = bitcast i64* %5 to i8*
  %11 = bitcast %struct.point43* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 6, i1 false)
  %12 = load i64, i64* %5, align 8
  ret i64 %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo44(i64 %0) #0 {
  %2 = alloca %struct.point44, align 2
  %3 = alloca %struct.point44, align 2
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point44* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 6, i1 false)
  %8 = bitcast %struct.point44* %2 to i8*
  %9 = bitcast %struct.point44* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 6, i1 false)
  %10 = bitcast i64* %5 to i8*
  %11 = bitcast %struct.point44* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 6, i1 false)
  %12 = load i64, i64* %5, align 8
  ret i64 %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo45(i64 %0) #0 {
  %2 = alloca %struct.point45, align 2
  %3 = alloca %struct.point45, align 2
  %4 = bitcast %struct.point45* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point45* %2 to i8*
  %6 = bitcast %struct.point45* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point45* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo46(i64 %0) #0 {
  %2 = alloca %struct.point46, align 2
  %3 = alloca %struct.point46, align 2
  %4 = bitcast %struct.point46* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point46* %2 to i8*
  %6 = bitcast %struct.point46* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point46* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo47([2 x i64] %0) #0 {
  %2 = alloca %struct.point47, align 4
  %3 = alloca %struct.point47, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point47* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point47* %2 to i8*
  %9 = bitcast %struct.point47* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point47* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo48([2 x i64] %0) #0 {
  %2 = alloca %struct.point48, align 4
  %3 = alloca %struct.point48, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point48* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point48* %2 to i8*
  %9 = bitcast %struct.point48* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point48* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo49(i64 %0) #0 {
  %2 = alloca %struct.point49, align 2
  %3 = alloca %struct.point49, align 2
  %4 = bitcast %struct.point49* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point49* %2 to i8*
  %6 = bitcast %struct.point49* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point49* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo50([2 x i64] %0) #0 {
  %2 = alloca %struct.point50, align 2
  %3 = alloca %struct.point50, align 2
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point50* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 10, i1 false)
  %8 = bitcast %struct.point50* %2 to i8*
  %9 = bitcast %struct.point50* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 10, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point50* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 10, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo51([2 x i64] %0) #0 {
  %2 = alloca %struct.point51, align 2
  %3 = alloca %struct.point51, align 2
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point51* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 10, i1 false)
  %8 = bitcast %struct.point51* %2 to i8*
  %9 = bitcast %struct.point51* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 10, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point51* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 10, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo52([2 x i64] %0) #0 {
  %2 = alloca %struct.point52, align 4
  %3 = alloca %struct.point52, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point52* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point52* %2 to i8*
  %9 = bitcast %struct.point52* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point52* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo53([2 x i64] %0) #0 {
  %2 = alloca %struct.point53, align 4
  %3 = alloca %struct.point53, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point53* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point53* %2 to i8*
  %9 = bitcast %struct.point53* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point53* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo54([2 x i64] %0) #0 {
  %2 = alloca %struct.point54, align 4
  %3 = alloca %struct.point54, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point54* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point54* %2 to i8*
  %9 = bitcast %struct.point54* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point54* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo55([2 x i64] %0) #0 {
  %2 = alloca %struct.point55, align 4
  %3 = alloca %struct.point55, align 4
  %4 = bitcast %struct.point55* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point55* %2 to i8*
  %6 = bitcast %struct.point55* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point55* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo56([2 x i64] %0) #0 {
  %2 = alloca %struct.point56, align 4
  %3 = alloca %struct.point56, align 4
  %4 = bitcast %struct.point56* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point56* %2 to i8*
  %6 = bitcast %struct.point56* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point56* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo57([2 x i64] %0) #0 {
  %2 = alloca %struct.point57, align 4
  %3 = alloca %struct.point57, align 4
  %4 = bitcast %struct.point57* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point57* %2 to i8*
  %6 = bitcast %struct.point57* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point57* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo58([2 x i64] %0) #0 {
  %2 = alloca %struct.point58, align 4
  %3 = alloca %struct.point58, align 4
  %4 = bitcast %struct.point58* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point58* %2 to i8*
  %6 = bitcast %struct.point58* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point58* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo59([2 x i64] %0) #0 {
  %2 = alloca %struct.point59, align 4
  %3 = alloca %struct.point59, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point59* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point59* %2 to i8*
  %9 = bitcast %struct.point59* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point59* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo60([2 x i64] %0) #0 {
  %2 = alloca %struct.point60, align 4
  %3 = alloca %struct.point60, align 4
  %4 = bitcast %struct.point60* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point60* %2 to i8*
  %6 = bitcast %struct.point60* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point60* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo61([2 x i64] %0) #0 {
  %2 = alloca %struct.point61, align 4
  %3 = alloca %struct.point61, align 4
  %4 = bitcast %struct.point61* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point61* %2 to i8*
  %6 = bitcast %struct.point61* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point61* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo62([2 x i64] %0) #0 {
  %2 = alloca %struct.point62, align 4
  %3 = alloca %struct.point62, align 4
  %4 = bitcast %struct.point62* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point62* %2 to i8*
  %6 = bitcast %struct.point62* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point62* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo63([2 x i64] %0) #0 {
  %2 = alloca %struct.point63, align 4
  %3 = alloca %struct.point63, align 4
  %4 = bitcast %struct.point63* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point63* %2 to i8*
  %6 = bitcast %struct.point63* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point63* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo64(i64 %0) #0 {
  %2 = alloca %struct.point64, align 2
  %3 = alloca %struct.point64, align 2
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point64* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 6, i1 false)
  %8 = bitcast %struct.point64* %2 to i8*
  %9 = bitcast %struct.point64* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 6, i1 false)
  %10 = bitcast i64* %5 to i8*
  %11 = bitcast %struct.point64* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 6, i1 false)
  %12 = load i64, i64* %5, align 8
  ret i64 %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo65(i64 %0) #0 {
  %2 = alloca %struct.point65, align 2
  %3 = alloca %struct.point65, align 2
  %4 = bitcast %struct.point65* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point65* %2 to i8*
  %6 = bitcast %struct.point65* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point65* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo66(i64 %0) #0 {
  %2 = alloca %struct.point66, align 2
  %3 = alloca %struct.point66, align 2
  %4 = bitcast %struct.point66* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point66* %2 to i8*
  %6 = bitcast %struct.point66* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point66* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo67([2 x i64] %0) #0 {
  %2 = alloca %struct.point67, align 2
  %3 = alloca %struct.point67, align 2
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point67* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 10, i1 false)
  %8 = bitcast %struct.point67* %2 to i8*
  %9 = bitcast %struct.point67* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 10, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point67* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 10, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo68([2 x i64] %0) #0 {
  %2 = alloca %struct.point68, align 4
  %3 = alloca %struct.point68, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point68* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point68* %2 to i8*
  %9 = bitcast %struct.point68* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point68* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo69([2 x i64] %0) #0 {
  %2 = alloca %struct.point69, align 4
  %3 = alloca %struct.point69, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point69* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point69* %2 to i8*
  %9 = bitcast %struct.point69* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point69* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo70(i64 %0) #0 {
  %2 = alloca %struct.point70, align 2
  %3 = alloca %struct.point70, align 2
  %4 = bitcast %struct.point70* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point70* %2 to i8*
  %6 = bitcast %struct.point70* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point70* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo71([2 x i64] %0) #0 {
  %2 = alloca %struct.point71, align 2
  %3 = alloca %struct.point71, align 2
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point71* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 10, i1 false)
  %8 = bitcast %struct.point71* %2 to i8*
  %9 = bitcast %struct.point71* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 10, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point71* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 10, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo72([2 x i64] %0) #0 {
  %2 = alloca %struct.point72, align 2
  %3 = alloca %struct.point72, align 2
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point72* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 10, i1 false)
  %8 = bitcast %struct.point72* %2 to i8*
  %9 = bitcast %struct.point72* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 10, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point72* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 10, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo73([2 x i64] %0) #0 {
  %2 = alloca %struct.point73, align 4
  %3 = alloca %struct.point73, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point73* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point73* %2 to i8*
  %9 = bitcast %struct.point73* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point73* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo74([2 x i64] %0) #0 {
  %2 = alloca %struct.point74, align 4
  %3 = alloca %struct.point74, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point74* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point74* %2 to i8*
  %9 = bitcast %struct.point74* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point74* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo75([2 x i64] %0) #0 {
  %2 = alloca %struct.point75, align 4
  %3 = alloca %struct.point75, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point75* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point75* %2 to i8*
  %9 = bitcast %struct.point75* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point75* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo76([2 x i64] %0) #0 {
  %2 = alloca %struct.point76, align 4
  %3 = alloca %struct.point76, align 4
  %4 = bitcast %struct.point76* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point76* %2 to i8*
  %6 = bitcast %struct.point76* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point76* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo77([2 x i64] %0) #0 {
  %2 = alloca %struct.point77, align 4
  %3 = alloca %struct.point77, align 4
  %4 = bitcast %struct.point77* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point77* %2 to i8*
  %6 = bitcast %struct.point77* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point77* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo78([2 x i64] %0) #0 {
  %2 = alloca %struct.point78, align 4
  %3 = alloca %struct.point78, align 4
  %4 = bitcast %struct.point78* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point78* %2 to i8*
  %6 = bitcast %struct.point78* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point78* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo79([2 x i64] %0) #0 {
  %2 = alloca %struct.point79, align 4
  %3 = alloca %struct.point79, align 4
  %4 = bitcast %struct.point79* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point79* %2 to i8*
  %6 = bitcast %struct.point79* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point79* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo80([2 x i64] %0) #0 {
  %2 = alloca %struct.point80, align 4
  %3 = alloca %struct.point80, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point80* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point80* %2 to i8*
  %9 = bitcast %struct.point80* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point80* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo81([2 x i64] %0) #0 {
  %2 = alloca %struct.point81, align 4
  %3 = alloca %struct.point81, align 4
  %4 = bitcast %struct.point81* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point81* %2 to i8*
  %6 = bitcast %struct.point81* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point81* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo82([2 x i64] %0) #0 {
  %2 = alloca %struct.point82, align 4
  %3 = alloca %struct.point82, align 4
  %4 = bitcast %struct.point82* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point82* %2 to i8*
  %6 = bitcast %struct.point82* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point82* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo83([2 x i64] %0) #0 {
  %2 = alloca %struct.point83, align 4
  %3 = alloca %struct.point83, align 4
  %4 = bitcast %struct.point83* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point83* %2 to i8*
  %6 = bitcast %struct.point83* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point83* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo84([2 x i64] %0) #0 {
  %2 = alloca %struct.point84, align 4
  %3 = alloca %struct.point84, align 4
  %4 = bitcast %struct.point84* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point84* %2 to i8*
  %6 = bitcast %struct.point84* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point84* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo85(i64 %0) #0 {
  %2 = alloca %struct.point85, align 4
  %3 = alloca %struct.point85, align 4
  %4 = bitcast %struct.point85* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point85* %2 to i8*
  %6 = bitcast %struct.point85* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point85* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo86([2 x i64] %0) #0 {
  %2 = alloca %struct.point86, align 4
  %3 = alloca %struct.point86, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point86* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point86* %2 to i8*
  %9 = bitcast %struct.point86* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point86* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo87([2 x i64] %0) #0 {
  %2 = alloca %struct.point87, align 4
  %3 = alloca %struct.point87, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point87* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point87* %2 to i8*
  %9 = bitcast %struct.point87* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point87* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo88([2 x i64] %0) #0 {
  %2 = alloca %struct.point88, align 4
  %3 = alloca %struct.point88, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point88* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point88* %2 to i8*
  %9 = bitcast %struct.point88* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point88* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo89([2 x i64] %0) #0 {
  %2 = alloca %struct.point89, align 4
  %3 = alloca %struct.point89, align 4
  %4 = bitcast %struct.point89* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point89* %2 to i8*
  %6 = bitcast %struct.point89* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point89* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo90([2 x i64] %0) #0 {
  %2 = alloca %struct.point90, align 4
  %3 = alloca %struct.point90, align 4
  %4 = bitcast %struct.point90* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point90* %2 to i8*
  %6 = bitcast %struct.point90* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point90* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo91([2 x i64] %0) #0 {
  %2 = alloca %struct.point91, align 4
  %3 = alloca %struct.point91, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point91* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point91* %2 to i8*
  %9 = bitcast %struct.point91* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point91* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo92([2 x i64] %0) #0 {
  %2 = alloca %struct.point92, align 4
  %3 = alloca %struct.point92, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point92* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point92* %2 to i8*
  %9 = bitcast %struct.point92* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point92* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo93([2 x i64] %0) #0 {
  %2 = alloca %struct.point93, align 4
  %3 = alloca %struct.point93, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point93* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point93* %2 to i8*
  %9 = bitcast %struct.point93* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point93* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo94([2 x i64] %0) #0 {
  %2 = alloca %struct.point94, align 4
  %3 = alloca %struct.point94, align 4
  %4 = bitcast %struct.point94* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point94* %2 to i8*
  %6 = bitcast %struct.point94* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point94* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo95([2 x i64] %0) #0 {
  %2 = alloca %struct.point95, align 4
  %3 = alloca %struct.point95, align 4
  %4 = bitcast %struct.point95* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point95* %2 to i8*
  %6 = bitcast %struct.point95* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point95* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo96([2 x i64] %0) #0 {
  %2 = alloca %struct.point96, align 4
  %3 = alloca %struct.point96, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point96* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point96* %2 to i8*
  %9 = bitcast %struct.point96* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point96* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo97([2 x i64] %0) #0 {
  %2 = alloca %struct.point97, align 4
  %3 = alloca %struct.point97, align 4
  %4 = bitcast %struct.point97* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point97* %2 to i8*
  %6 = bitcast %struct.point97* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point97* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo98([2 x i64] %0) #0 {
  %2 = alloca %struct.point98, align 4
  %3 = alloca %struct.point98, align 4
  %4 = bitcast %struct.point98* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point98* %2 to i8*
  %6 = bitcast %struct.point98* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point98* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo99([2 x i64] %0) #0 {
  %2 = alloca %struct.point99, align 4
  %3 = alloca %struct.point99, align 4
  %4 = bitcast %struct.point99* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point99* %2 to i8*
  %6 = bitcast %struct.point99* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point99* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo100([2 x i64] %0) #0 {
  %2 = alloca %struct.point100, align 4
  %3 = alloca %struct.point100, align 4
  %4 = bitcast %struct.point100* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point100* %2 to i8*
  %6 = bitcast %struct.point100* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point100* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo101([2 x i64] %0) #0 {
  %2 = alloca %struct.point101, align 4
  %3 = alloca %struct.point101, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point101* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point101* %2 to i8*
  %9 = bitcast %struct.point101* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point101* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo102([2 x i64] %0) #0 {
  %2 = alloca %struct.point102, align 4
  %3 = alloca %struct.point102, align 4
  %4 = bitcast %struct.point102* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point102* %2 to i8*
  %6 = bitcast %struct.point102* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point102* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo103([2 x i64] %0) #0 {
  %2 = alloca %struct.point103, align 4
  %3 = alloca %struct.point103, align 4
  %4 = bitcast %struct.point103* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point103* %2 to i8*
  %6 = bitcast %struct.point103* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point103* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo104([2 x i64] %0) #0 {
  %2 = alloca %struct.point104, align 4
  %3 = alloca %struct.point104, align 4
  %4 = bitcast %struct.point104* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point104* %2 to i8*
  %6 = bitcast %struct.point104* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point104* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo105([2 x i64] %0) #0 {
  %2 = alloca %struct.point105, align 4
  %3 = alloca %struct.point105, align 4
  %4 = bitcast %struct.point105* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point105* %2 to i8*
  %6 = bitcast %struct.point105* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point105* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo106(i64 %0) #0 {
  %2 = alloca %struct.point106, align 4
  %3 = alloca %struct.point106, align 4
  %4 = bitcast %struct.point106* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point106* %2 to i8*
  %6 = bitcast %struct.point106* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point106* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo107([2 x i64] %0) #0 {
  %2 = alloca %struct.point107, align 4
  %3 = alloca %struct.point107, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point107* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point107* %2 to i8*
  %9 = bitcast %struct.point107* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point107* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo108([2 x i64] %0) #0 {
  %2 = alloca %struct.point108, align 4
  %3 = alloca %struct.point108, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point108* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point108* %2 to i8*
  %9 = bitcast %struct.point108* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point108* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo109([2 x i64] %0) #0 {
  %2 = alloca %struct.point109, align 4
  %3 = alloca %struct.point109, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point109* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point109* %2 to i8*
  %9 = bitcast %struct.point109* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point109* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo110([2 x i64] %0) #0 {
  %2 = alloca %struct.point110, align 4
  %3 = alloca %struct.point110, align 4
  %4 = bitcast %struct.point110* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point110* %2 to i8*
  %6 = bitcast %struct.point110* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point110* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo111([2 x i64] %0) #0 {
  %2 = alloca %struct.point111, align 4
  %3 = alloca %struct.point111, align 4
  %4 = bitcast %struct.point111* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point111* %2 to i8*
  %6 = bitcast %struct.point111* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point111* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo112([2 x i64] %0) #0 {
  %2 = alloca %struct.point112, align 4
  %3 = alloca %struct.point112, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point112* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point112* %2 to i8*
  %9 = bitcast %struct.point112* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point112* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo113([2 x i64] %0) #0 {
  %2 = alloca %struct.point113, align 4
  %3 = alloca %struct.point113, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point113* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point113* %2 to i8*
  %9 = bitcast %struct.point113* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point113* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo114([2 x i64] %0) #0 {
  %2 = alloca %struct.point114, align 4
  %3 = alloca %struct.point114, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point114* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point114* %2 to i8*
  %9 = bitcast %struct.point114* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point114* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo115([2 x i64] %0) #0 {
  %2 = alloca %struct.point115, align 4
  %3 = alloca %struct.point115, align 4
  %4 = bitcast %struct.point115* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point115* %2 to i8*
  %6 = bitcast %struct.point115* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point115* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo116([2 x i64] %0) #0 {
  %2 = alloca %struct.point116, align 4
  %3 = alloca %struct.point116, align 4
  %4 = bitcast %struct.point116* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point116* %2 to i8*
  %6 = bitcast %struct.point116* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point116* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo117([2 x i64] %0) #0 {
  %2 = alloca %struct.point117, align 4
  %3 = alloca %struct.point117, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point117* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point117* %2 to i8*
  %9 = bitcast %struct.point117* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point117* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo118([2 x i64] %0) #0 {
  %2 = alloca %struct.point118, align 4
  %3 = alloca %struct.point118, align 4
  %4 = bitcast %struct.point118* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point118* %2 to i8*
  %6 = bitcast %struct.point118* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point118* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo119([2 x i64] %0) #0 {
  %2 = alloca %struct.point119, align 4
  %3 = alloca %struct.point119, align 4
  %4 = bitcast %struct.point119* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point119* %2 to i8*
  %6 = bitcast %struct.point119* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point119* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo120([2 x i64] %0) #0 {
  %2 = alloca %struct.point120, align 4
  %3 = alloca %struct.point120, align 4
  %4 = bitcast %struct.point120* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point120* %2 to i8*
  %6 = bitcast %struct.point120* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point120* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo121([2 x i64] %0) #0 {
  %2 = alloca %struct.point121, align 4
  %3 = alloca %struct.point121, align 4
  %4 = bitcast %struct.point121* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point121* %2 to i8*
  %6 = bitcast %struct.point121* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point121* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo122([2 x i64] %0) #0 {
  %2 = alloca %struct.point122, align 4
  %3 = alloca %struct.point122, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point122* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point122* %2 to i8*
  %9 = bitcast %struct.point122* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point122* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo123([2 x i64] %0) #0 {
  %2 = alloca %struct.point123, align 4
  %3 = alloca %struct.point123, align 4
  %4 = bitcast %struct.point123* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point123* %2 to i8*
  %6 = bitcast %struct.point123* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point123* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo124([2 x i64] %0) #0 {
  %2 = alloca %struct.point124, align 4
  %3 = alloca %struct.point124, align 4
  %4 = bitcast %struct.point124* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point124* %2 to i8*
  %6 = bitcast %struct.point124* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point124* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo125([2 x i64] %0) #0 {
  %2 = alloca %struct.point125, align 4
  %3 = alloca %struct.point125, align 4
  %4 = bitcast %struct.point125* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point125* %2 to i8*
  %6 = bitcast %struct.point125* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point125* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo126([2 x i64] %0) #0 {
  %2 = alloca %struct.point126, align 4
  %3 = alloca %struct.point126, align 4
  %4 = bitcast %struct.point126* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point126* %2 to i8*
  %6 = bitcast %struct.point126* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point126* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo127([2 x i64] %0) #0 {
  %2 = alloca %struct.point127, align 4
  %3 = alloca %struct.point127, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point127* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point127* %2 to i8*
  %9 = bitcast %struct.point127* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point127* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo128([2 x i64] %0) #0 {
  %2 = alloca %struct.point128, align 4
  %3 = alloca %struct.point128, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point128* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point128* %2 to i8*
  %9 = bitcast %struct.point128* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point128* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo129([2 x i64] %0) #0 {
  %2 = alloca %struct.point129, align 4
  %3 = alloca %struct.point129, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point129* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point129* %2 to i8*
  %9 = bitcast %struct.point129* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point129* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo130([2 x i64] %0) #0 {
  %2 = alloca %struct.point130, align 4
  %3 = alloca %struct.point130, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point130* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point130* %2 to i8*
  %9 = bitcast %struct.point130* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point130* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo131([2 x i64] %0) #0 {
  %2 = alloca %struct.point131, align 4
  %3 = alloca %struct.point131, align 4
  %4 = bitcast %struct.point131* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point131* %2 to i8*
  %6 = bitcast %struct.point131* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point131* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo132([2 x i64] %0) #0 {
  %2 = alloca %struct.point132, align 4
  %3 = alloca %struct.point132, align 4
  %4 = bitcast %struct.point132* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point132* %2 to i8*
  %6 = bitcast %struct.point132* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point132* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo133([2 x i64] %0) #0 {
  %2 = alloca %struct.point133, align 4
  %3 = alloca %struct.point133, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point133* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point133* %2 to i8*
  %9 = bitcast %struct.point133* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point133* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo134([2 x i64] %0) #0 {
  %2 = alloca %struct.point134, align 4
  %3 = alloca %struct.point134, align 4
  %4 = bitcast %struct.point134* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point134* %2 to i8*
  %6 = bitcast %struct.point134* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point134* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo135([2 x i64] %0) #0 {
  %2 = alloca %struct.point135, align 4
  %3 = alloca %struct.point135, align 4
  %4 = bitcast %struct.point135* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point135* %2 to i8*
  %6 = bitcast %struct.point135* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point135* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo136([2 x i64] %0) #0 {
  %2 = alloca %struct.point136, align 4
  %3 = alloca %struct.point136, align 4
  %4 = bitcast %struct.point136* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point136* %2 to i8*
  %6 = bitcast %struct.point136* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point136* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo137([2 x i64] %0) #0 {
  %2 = alloca %struct.point137, align 4
  %3 = alloca %struct.point137, align 4
  %4 = bitcast %struct.point137* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point137* %2 to i8*
  %6 = bitcast %struct.point137* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point137* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo138([2 x i64] %0) #0 {
  %2 = alloca %struct.point138, align 4
  %3 = alloca %struct.point138, align 4
  %4 = bitcast %struct.point138* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point138* %2 to i8*
  %6 = bitcast %struct.point138* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point138* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo139(%struct.point139* noalias sret(%struct.point139) align 4 %0, %struct.point139* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point139* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point139* %0 to i8*
  %6 = bitcast %struct.point139* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo140(%struct.point140* noalias sret(%struct.point140) align 4 %0, %struct.point140* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point140* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point140* %0 to i8*
  %6 = bitcast %struct.point140* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo141(%struct.point141* noalias sret(%struct.point141) align 4 %0, %struct.point141* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point141* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point141* %0 to i8*
  %6 = bitcast %struct.point141* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo142(%struct.point142* noalias sret(%struct.point142) align 4 %0, %struct.point142* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point142* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point142* %0 to i8*
  %6 = bitcast %struct.point142* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo143([2 x i64] %0) #0 {
  %2 = alloca %struct.point143, align 4
  %3 = alloca %struct.point143, align 4
  %4 = bitcast %struct.point143* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point143* %2 to i8*
  %6 = bitcast %struct.point143* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point143* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo144(%struct.point144* noalias sret(%struct.point144) align 4 %0, %struct.point144* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point144* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point144* %0 to i8*
  %6 = bitcast %struct.point144* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo145(%struct.point145* noalias sret(%struct.point145) align 4 %0, %struct.point145* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point145* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point145* %0 to i8*
  %6 = bitcast %struct.point145* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo146(%struct.point146* noalias sret(%struct.point146) align 4 %0, %struct.point146* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point146* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point146* %0 to i8*
  %6 = bitcast %struct.point146* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo147(%struct.point147* noalias sret(%struct.point147) align 4 %0, %struct.point147* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point147* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point147* %0 to i8*
  %6 = bitcast %struct.point147* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo148([2 x i64] %0) #0 {
  %2 = alloca %struct.point148, align 4
  %3 = alloca %struct.point148, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point148* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point148* %2 to i8*
  %9 = bitcast %struct.point148* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point148* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo149([2 x i64] %0) #0 {
  %2 = alloca %struct.point149, align 4
  %3 = alloca %struct.point149, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point149* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point149* %2 to i8*
  %9 = bitcast %struct.point149* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point149* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo150([2 x i64] %0) #0 {
  %2 = alloca %struct.point150, align 4
  %3 = alloca %struct.point150, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point150* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point150* %2 to i8*
  %9 = bitcast %struct.point150* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point150* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo151([2 x i64] %0) #0 {
  %2 = alloca %struct.point151, align 4
  %3 = alloca %struct.point151, align 4
  %4 = bitcast %struct.point151* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point151* %2 to i8*
  %6 = bitcast %struct.point151* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point151* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo152([2 x i64] %0) #0 {
  %2 = alloca %struct.point152, align 4
  %3 = alloca %struct.point152, align 4
  %4 = bitcast %struct.point152* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point152* %2 to i8*
  %6 = bitcast %struct.point152* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point152* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo153([2 x i64] %0) #0 {
  %2 = alloca %struct.point153, align 4
  %3 = alloca %struct.point153, align 4
  %4 = bitcast %struct.point153* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point153* %2 to i8*
  %6 = bitcast %struct.point153* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point153* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo154([2 x i64] %0) #0 {
  %2 = alloca %struct.point154, align 4
  %3 = alloca %struct.point154, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point154* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point154* %2 to i8*
  %9 = bitcast %struct.point154* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point154* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo155([2 x i64] %0) #0 {
  %2 = alloca %struct.point155, align 4
  %3 = alloca %struct.point155, align 4
  %4 = bitcast %struct.point155* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point155* %2 to i8*
  %6 = bitcast %struct.point155* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point155* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo156([2 x i64] %0) #0 {
  %2 = alloca %struct.point156, align 4
  %3 = alloca %struct.point156, align 4
  %4 = bitcast %struct.point156* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point156* %2 to i8*
  %6 = bitcast %struct.point156* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point156* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo157([2 x i64] %0) #0 {
  %2 = alloca %struct.point157, align 4
  %3 = alloca %struct.point157, align 4
  %4 = bitcast %struct.point157* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point157* %2 to i8*
  %6 = bitcast %struct.point157* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point157* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo158([2 x i64] %0) #0 {
  %2 = alloca %struct.point158, align 4
  %3 = alloca %struct.point158, align 4
  %4 = bitcast %struct.point158* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point158* %2 to i8*
  %6 = bitcast %struct.point158* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point158* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo159([2 x i64] %0) #0 {
  %2 = alloca %struct.point159, align 4
  %3 = alloca %struct.point159, align 4
  %4 = bitcast %struct.point159* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point159* %2 to i8*
  %6 = bitcast %struct.point159* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point159* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo160(%struct.point160* noalias sret(%struct.point160) align 4 %0, %struct.point160* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point160* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point160* %0 to i8*
  %6 = bitcast %struct.point160* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo161(%struct.point161* noalias sret(%struct.point161) align 4 %0, %struct.point161* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point161* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point161* %0 to i8*
  %6 = bitcast %struct.point161* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo162(%struct.point162* noalias sret(%struct.point162) align 4 %0, %struct.point162* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point162* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point162* %0 to i8*
  %6 = bitcast %struct.point162* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo163(%struct.point163* noalias sret(%struct.point163) align 4 %0, %struct.point163* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point163* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point163* %0 to i8*
  %6 = bitcast %struct.point163* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo164([2 x i64] %0) #0 {
  %2 = alloca %struct.point164, align 4
  %3 = alloca %struct.point164, align 4
  %4 = bitcast %struct.point164* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point164* %2 to i8*
  %6 = bitcast %struct.point164* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point164* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo165(%struct.point165* noalias sret(%struct.point165) align 4 %0, %struct.point165* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point165* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point165* %0 to i8*
  %6 = bitcast %struct.point165* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo166(%struct.point166* noalias sret(%struct.point166) align 4 %0, %struct.point166* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point166* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point166* %0 to i8*
  %6 = bitcast %struct.point166* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo167(%struct.point167* noalias sret(%struct.point167) align 4 %0, %struct.point167* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point167* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point167* %0 to i8*
  %6 = bitcast %struct.point167* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo168(%struct.point168* noalias sret(%struct.point168) align 4 %0, %struct.point168* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point168* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point168* %0 to i8*
  %6 = bitcast %struct.point168* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo169([2 x i64] %0) #0 {
  %2 = alloca %struct.point169, align 4
  %3 = alloca %struct.point169, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point169* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point169* %2 to i8*
  %9 = bitcast %struct.point169* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point169* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo170([2 x i64] %0) #0 {
  %2 = alloca %struct.point170, align 4
  %3 = alloca %struct.point170, align 4
  %4 = bitcast %struct.point170* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point170* %2 to i8*
  %6 = bitcast %struct.point170* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point170* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo171([2 x i64] %0) #0 {
  %2 = alloca %struct.point171, align 4
  %3 = alloca %struct.point171, align 4
  %4 = bitcast %struct.point171* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point171* %2 to i8*
  %6 = bitcast %struct.point171* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point171* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo172([2 x i64] %0) #0 {
  %2 = alloca %struct.point172, align 4
  %3 = alloca %struct.point172, align 4
  %4 = bitcast %struct.point172* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point172* %2 to i8*
  %6 = bitcast %struct.point172* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point172* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo173(%struct.point173* noalias sret(%struct.point173) align 4 %0, %struct.point173* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point173* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point173* %0 to i8*
  %6 = bitcast %struct.point173* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo174(%struct.point174* noalias sret(%struct.point174) align 4 %0, %struct.point174* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point174* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point174* %0 to i8*
  %6 = bitcast %struct.point174* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo175([2 x i64] %0) #0 {
  %2 = alloca %struct.point175, align 4
  %3 = alloca %struct.point175, align 4
  %4 = bitcast %struct.point175* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point175* %2 to i8*
  %6 = bitcast %struct.point175* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point175* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo176([2 x i64] %0) #0 {
  %2 = alloca %struct.point176, align 4
  %3 = alloca %struct.point176, align 4
  %4 = bitcast %struct.point176* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point176* %2 to i8*
  %6 = bitcast %struct.point176* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point176* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo177([2 x i64] %0) #0 {
  %2 = alloca %struct.point177, align 4
  %3 = alloca %struct.point177, align 4
  %4 = bitcast %struct.point177* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point177* %2 to i8*
  %6 = bitcast %struct.point177* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point177* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo178(%struct.point178* noalias sret(%struct.point178) align 4 %0, %struct.point178* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point178* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point178* %0 to i8*
  %6 = bitcast %struct.point178* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo179(%struct.point179* noalias sret(%struct.point179) align 4 %0, %struct.point179* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point179* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point179* %0 to i8*
  %6 = bitcast %struct.point179* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo180([2 x i64] %0) #0 {
  %2 = alloca %struct.point180, align 4
  %3 = alloca %struct.point180, align 4
  %4 = bitcast %struct.point180* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point180* %2 to i8*
  %6 = bitcast %struct.point180* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point180* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo181(%struct.point181* noalias sret(%struct.point181) align 4 %0, %struct.point181* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point181* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point181* %0 to i8*
  %6 = bitcast %struct.point181* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo182(%struct.point182* noalias sret(%struct.point182) align 4 %0, %struct.point182* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point182* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point182* %0 to i8*
  %6 = bitcast %struct.point182* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo183(%struct.point183* noalias sret(%struct.point183) align 4 %0, %struct.point183* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point183* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point183* %0 to i8*
  %6 = bitcast %struct.point183* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo184(%struct.point184* noalias sret(%struct.point184) align 4 %0, %struct.point184* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point184* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point184* %0 to i8*
  %6 = bitcast %struct.point184* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo185([2 x i64] %0) #0 {
  %2 = alloca %struct.point185, align 4
  %3 = alloca %struct.point185, align 4
  %4 = bitcast %struct.point185* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point185* %2 to i8*
  %6 = bitcast %struct.point185* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point185* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo186(%struct.point186* noalias sret(%struct.point186) align 4 %0, %struct.point186* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point186* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point186* %0 to i8*
  %6 = bitcast %struct.point186* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo187(%struct.point187* noalias sret(%struct.point187) align 4 %0, %struct.point187* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point187* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point187* %0 to i8*
  %6 = bitcast %struct.point187* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo188(%struct.point188* noalias sret(%struct.point188) align 4 %0, %struct.point188* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point188* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point188* %0 to i8*
  %6 = bitcast %struct.point188* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo189(%struct.point189* noalias sret(%struct.point189) align 4 %0, %struct.point189* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point189* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point189* %0 to i8*
  %6 = bitcast %struct.point189* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo190([2 x i64] %0) #0 {
  %2 = alloca %struct.point190, align 4
  %3 = alloca %struct.point190, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point190* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point190* %2 to i8*
  %9 = bitcast %struct.point190* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point190* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo191([2 x i64] %0) #0 {
  %2 = alloca %struct.point191, align 4
  %3 = alloca %struct.point191, align 4
  %4 = bitcast %struct.point191* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point191* %2 to i8*
  %6 = bitcast %struct.point191* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point191* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo192([2 x i64] %0) #0 {
  %2 = alloca %struct.point192, align 4
  %3 = alloca %struct.point192, align 4
  %4 = bitcast %struct.point192* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point192* %2 to i8*
  %6 = bitcast %struct.point192* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point192* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo193([2 x i64] %0) #0 {
  %2 = alloca %struct.point193, align 4
  %3 = alloca %struct.point193, align 4
  %4 = bitcast %struct.point193* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point193* %2 to i8*
  %6 = bitcast %struct.point193* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point193* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo194(%struct.point194* noalias sret(%struct.point194) align 4 %0, %struct.point194* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point194* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point194* %0 to i8*
  %6 = bitcast %struct.point194* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo195(%struct.point195* noalias sret(%struct.point195) align 4 %0, %struct.point195* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point195* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point195* %0 to i8*
  %6 = bitcast %struct.point195* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo196([2 x i64] %0) #0 {
  %2 = alloca %struct.point196, align 4
  %3 = alloca %struct.point196, align 4
  %4 = bitcast %struct.point196* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point196* %2 to i8*
  %6 = bitcast %struct.point196* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point196* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo197([2 x i64] %0) #0 {
  %2 = alloca %struct.point197, align 4
  %3 = alloca %struct.point197, align 4
  %4 = bitcast %struct.point197* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point197* %2 to i8*
  %6 = bitcast %struct.point197* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point197* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo198([2 x i64] %0) #0 {
  %2 = alloca %struct.point198, align 4
  %3 = alloca %struct.point198, align 4
  %4 = bitcast %struct.point198* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point198* %2 to i8*
  %6 = bitcast %struct.point198* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point198* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo199(%struct.point199* noalias sret(%struct.point199) align 4 %0, %struct.point199* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point199* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point199* %0 to i8*
  %6 = bitcast %struct.point199* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo200(%struct.point200* noalias sret(%struct.point200) align 4 %0, %struct.point200* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point200* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point200* %0 to i8*
  %6 = bitcast %struct.point200* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo201([2 x i64] %0) #0 {
  %2 = alloca %struct.point201, align 4
  %3 = alloca %struct.point201, align 4
  %4 = bitcast %struct.point201* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point201* %2 to i8*
  %6 = bitcast %struct.point201* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point201* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo202(%struct.point202* noalias sret(%struct.point202) align 4 %0, %struct.point202* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point202* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point202* %0 to i8*
  %6 = bitcast %struct.point202* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo203(%struct.point203* noalias sret(%struct.point203) align 4 %0, %struct.point203* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point203* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point203* %0 to i8*
  %6 = bitcast %struct.point203* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo204(%struct.point204* noalias sret(%struct.point204) align 4 %0, %struct.point204* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point204* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point204* %0 to i8*
  %6 = bitcast %struct.point204* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo205(%struct.point205* noalias sret(%struct.point205) align 4 %0, %struct.point205* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point205* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point205* %0 to i8*
  %6 = bitcast %struct.point205* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo206([2 x i64] %0) #0 {
  %2 = alloca %struct.point206, align 4
  %3 = alloca %struct.point206, align 4
  %4 = bitcast %struct.point206* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point206* %2 to i8*
  %6 = bitcast %struct.point206* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point206* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo207(%struct.point207* noalias sret(%struct.point207) align 4 %0, %struct.point207* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point207* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point207* %0 to i8*
  %6 = bitcast %struct.point207* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo208(%struct.point208* noalias sret(%struct.point208) align 4 %0, %struct.point208* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point208* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point208* %0 to i8*
  %6 = bitcast %struct.point208* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo209(%struct.point209* noalias sret(%struct.point209) align 4 %0, %struct.point209* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point209* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point209* %0 to i8*
  %6 = bitcast %struct.point209* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo210(%struct.point210* noalias sret(%struct.point210) align 4 %0, %struct.point210* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point210* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point210* %0 to i8*
  %6 = bitcast %struct.point210* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo211([2 x i64] %0) #0 {
  %2 = alloca %struct.point211, align 4
  %3 = alloca %struct.point211, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point211* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point211* %2 to i8*
  %9 = bitcast %struct.point211* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point211* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo212([2 x i64] %0) #0 {
  %2 = alloca %struct.point212, align 4
  %3 = alloca %struct.point212, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point212* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point212* %2 to i8*
  %9 = bitcast %struct.point212* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point212* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo213([2 x i64] %0) #0 {
  %2 = alloca %struct.point213, align 4
  %3 = alloca %struct.point213, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point213* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point213* %2 to i8*
  %9 = bitcast %struct.point213* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point213* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo214([2 x i64] %0) #0 {
  %2 = alloca %struct.point214, align 4
  %3 = alloca %struct.point214, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point214* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point214* %2 to i8*
  %9 = bitcast %struct.point214* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point214* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo215([2 x i64] %0) #0 {
  %2 = alloca %struct.point215, align 4
  %3 = alloca %struct.point215, align 4
  %4 = bitcast %struct.point215* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point215* %2 to i8*
  %6 = bitcast %struct.point215* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point215* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo216([2 x i64] %0) #0 {
  %2 = alloca %struct.point216, align 4
  %3 = alloca %struct.point216, align 4
  %4 = bitcast %struct.point216* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point216* %2 to i8*
  %6 = bitcast %struct.point216* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point216* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo217([2 x i64] %0) #0 {
  %2 = alloca %struct.point217, align 4
  %3 = alloca %struct.point217, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point217* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point217* %2 to i8*
  %9 = bitcast %struct.point217* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point217* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo218([2 x i64] %0) #0 {
  %2 = alloca %struct.point218, align 4
  %3 = alloca %struct.point218, align 4
  %4 = bitcast %struct.point218* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point218* %2 to i8*
  %6 = bitcast %struct.point218* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point218* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo219([2 x i64] %0) #0 {
  %2 = alloca %struct.point219, align 4
  %3 = alloca %struct.point219, align 4
  %4 = bitcast %struct.point219* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point219* %2 to i8*
  %6 = bitcast %struct.point219* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point219* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo220([2 x i64] %0) #0 {
  %2 = alloca %struct.point220, align 4
  %3 = alloca %struct.point220, align 4
  %4 = bitcast %struct.point220* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point220* %2 to i8*
  %6 = bitcast %struct.point220* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point220* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo221([2 x i64] %0) #0 {
  %2 = alloca %struct.point221, align 4
  %3 = alloca %struct.point221, align 4
  %4 = bitcast %struct.point221* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point221* %2 to i8*
  %6 = bitcast %struct.point221* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point221* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo222([2 x i64] %0) #0 {
  %2 = alloca %struct.point222, align 4
  %3 = alloca %struct.point222, align 4
  %4 = bitcast %struct.point222* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point222* %2 to i8*
  %6 = bitcast %struct.point222* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point222* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo223(%struct.point223* noalias sret(%struct.point223) align 4 %0, %struct.point223* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point223* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point223* %0 to i8*
  %6 = bitcast %struct.point223* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo224(%struct.point224* noalias sret(%struct.point224) align 4 %0, %struct.point224* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point224* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point224* %0 to i8*
  %6 = bitcast %struct.point224* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo225(%struct.point225* noalias sret(%struct.point225) align 4 %0, %struct.point225* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point225* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point225* %0 to i8*
  %6 = bitcast %struct.point225* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo226(%struct.point226* noalias sret(%struct.point226) align 4 %0, %struct.point226* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point226* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point226* %0 to i8*
  %6 = bitcast %struct.point226* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo227([2 x i64] %0) #0 {
  %2 = alloca %struct.point227, align 4
  %3 = alloca %struct.point227, align 4
  %4 = bitcast %struct.point227* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point227* %2 to i8*
  %6 = bitcast %struct.point227* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point227* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo228(%struct.point228* noalias sret(%struct.point228) align 4 %0, %struct.point228* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point228* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point228* %0 to i8*
  %6 = bitcast %struct.point228* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo229(%struct.point229* noalias sret(%struct.point229) align 4 %0, %struct.point229* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point229* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point229* %0 to i8*
  %6 = bitcast %struct.point229* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo230(%struct.point230* noalias sret(%struct.point230) align 4 %0, %struct.point230* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point230* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point230* %0 to i8*
  %6 = bitcast %struct.point230* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo231(%struct.point231* noalias sret(%struct.point231) align 4 %0, %struct.point231* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point231* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point231* %0 to i8*
  %6 = bitcast %struct.point231* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo232([2 x i64] %0) #0 {
  %2 = alloca %struct.point232, align 4
  %3 = alloca %struct.point232, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point232* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point232* %2 to i8*
  %9 = bitcast %struct.point232* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point232* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo233([2 x i64] %0) #0 {
  %2 = alloca %struct.point233, align 4
  %3 = alloca %struct.point233, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point233* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point233* %2 to i8*
  %9 = bitcast %struct.point233* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point233* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo234([2 x i64] %0) #0 {
  %2 = alloca %struct.point234, align 4
  %3 = alloca %struct.point234, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point234* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point234* %2 to i8*
  %9 = bitcast %struct.point234* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point234* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo235([2 x i64] %0) #0 {
  %2 = alloca %struct.point235, align 4
  %3 = alloca %struct.point235, align 4
  %4 = bitcast %struct.point235* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point235* %2 to i8*
  %6 = bitcast %struct.point235* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point235* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo236([2 x i64] %0) #0 {
  %2 = alloca %struct.point236, align 4
  %3 = alloca %struct.point236, align 4
  %4 = bitcast %struct.point236* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point236* %2 to i8*
  %6 = bitcast %struct.point236* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point236* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo237([2 x i64] %0) #0 {
  %2 = alloca %struct.point237, align 4
  %3 = alloca %struct.point237, align 4
  %4 = bitcast %struct.point237* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point237* %2 to i8*
  %6 = bitcast %struct.point237* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point237* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo238([2 x i64] %0) #0 {
  %2 = alloca %struct.point238, align 4
  %3 = alloca %struct.point238, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point238* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point238* %2 to i8*
  %9 = bitcast %struct.point238* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point238* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo239([2 x i64] %0) #0 {
  %2 = alloca %struct.point239, align 4
  %3 = alloca %struct.point239, align 4
  %4 = bitcast %struct.point239* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point239* %2 to i8*
  %6 = bitcast %struct.point239* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point239* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo240([2 x i64] %0) #0 {
  %2 = alloca %struct.point240, align 4
  %3 = alloca %struct.point240, align 4
  %4 = bitcast %struct.point240* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point240* %2 to i8*
  %6 = bitcast %struct.point240* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point240* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo241([2 x i64] %0) #0 {
  %2 = alloca %struct.point241, align 4
  %3 = alloca %struct.point241, align 4
  %4 = bitcast %struct.point241* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point241* %2 to i8*
  %6 = bitcast %struct.point241* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point241* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo242([2 x i64] %0) #0 {
  %2 = alloca %struct.point242, align 4
  %3 = alloca %struct.point242, align 4
  %4 = bitcast %struct.point242* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point242* %2 to i8*
  %6 = bitcast %struct.point242* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point242* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo243([2 x i64] %0) #0 {
  %2 = alloca %struct.point243, align 4
  %3 = alloca %struct.point243, align 4
  %4 = bitcast %struct.point243* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point243* %2 to i8*
  %6 = bitcast %struct.point243* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point243* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo244(%struct.point244* noalias sret(%struct.point244) align 4 %0, %struct.point244* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point244* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point244* %0 to i8*
  %6 = bitcast %struct.point244* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo245(%struct.point245* noalias sret(%struct.point245) align 4 %0, %struct.point245* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point245* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point245* %0 to i8*
  %6 = bitcast %struct.point245* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo246(%struct.point246* noalias sret(%struct.point246) align 4 %0, %struct.point246* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point246* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point246* %0 to i8*
  %6 = bitcast %struct.point246* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo247(%struct.point247* noalias sret(%struct.point247) align 4 %0, %struct.point247* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point247* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point247* %0 to i8*
  %6 = bitcast %struct.point247* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo248([2 x i64] %0) #0 {
  %2 = alloca %struct.point248, align 4
  %3 = alloca %struct.point248, align 4
  %4 = bitcast %struct.point248* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point248* %2 to i8*
  %6 = bitcast %struct.point248* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point248* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo249(%struct.point249* noalias sret(%struct.point249) align 4 %0, %struct.point249* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point249* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point249* %0 to i8*
  %6 = bitcast %struct.point249* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo250(%struct.point250* noalias sret(%struct.point250) align 4 %0, %struct.point250* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point250* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point250* %0 to i8*
  %6 = bitcast %struct.point250* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo251(%struct.point251* noalias sret(%struct.point251) align 4 %0, %struct.point251* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point251* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point251* %0 to i8*
  %6 = bitcast %struct.point251* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo252(%struct.point252* noalias sret(%struct.point252) align 4 %0, %struct.point252* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point252* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point252* %0 to i8*
  %6 = bitcast %struct.point252* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo253([2 x i64] %0) #0 {
  %2 = alloca %struct.point253, align 4
  %3 = alloca %struct.point253, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point253* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point253* %2 to i8*
  %9 = bitcast %struct.point253* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point253* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo254([2 x i64] %0) #0 {
  %2 = alloca %struct.point254, align 4
  %3 = alloca %struct.point254, align 4
  %4 = bitcast %struct.point254* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point254* %2 to i8*
  %6 = bitcast %struct.point254* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point254* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo255([2 x i64] %0) #0 {
  %2 = alloca %struct.point255, align 4
  %3 = alloca %struct.point255, align 4
  %4 = bitcast %struct.point255* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point255* %2 to i8*
  %6 = bitcast %struct.point255* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point255* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo256([2 x i64] %0) #0 {
  %2 = alloca %struct.point256, align 4
  %3 = alloca %struct.point256, align 4
  %4 = bitcast %struct.point256* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point256* %2 to i8*
  %6 = bitcast %struct.point256* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point256* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo257(%struct.point257* noalias sret(%struct.point257) align 4 %0, %struct.point257* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point257* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point257* %0 to i8*
  %6 = bitcast %struct.point257* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo258(%struct.point258* noalias sret(%struct.point258) align 4 %0, %struct.point258* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point258* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point258* %0 to i8*
  %6 = bitcast %struct.point258* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo259([2 x i64] %0) #0 {
  %2 = alloca %struct.point259, align 4
  %3 = alloca %struct.point259, align 4
  %4 = bitcast %struct.point259* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point259* %2 to i8*
  %6 = bitcast %struct.point259* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point259* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo260([2 x i64] %0) #0 {
  %2 = alloca %struct.point260, align 4
  %3 = alloca %struct.point260, align 4
  %4 = bitcast %struct.point260* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point260* %2 to i8*
  %6 = bitcast %struct.point260* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point260* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo261([2 x i64] %0) #0 {
  %2 = alloca %struct.point261, align 4
  %3 = alloca %struct.point261, align 4
  %4 = bitcast %struct.point261* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point261* %2 to i8*
  %6 = bitcast %struct.point261* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point261* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo262(%struct.point262* noalias sret(%struct.point262) align 4 %0, %struct.point262* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point262* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point262* %0 to i8*
  %6 = bitcast %struct.point262* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo263(%struct.point263* noalias sret(%struct.point263) align 4 %0, %struct.point263* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point263* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point263* %0 to i8*
  %6 = bitcast %struct.point263* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo264([2 x i64] %0) #0 {
  %2 = alloca %struct.point264, align 4
  %3 = alloca %struct.point264, align 4
  %4 = bitcast %struct.point264* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point264* %2 to i8*
  %6 = bitcast %struct.point264* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point264* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo265(%struct.point265* noalias sret(%struct.point265) align 4 %0, %struct.point265* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point265* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point265* %0 to i8*
  %6 = bitcast %struct.point265* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo266(%struct.point266* noalias sret(%struct.point266) align 4 %0, %struct.point266* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point266* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point266* %0 to i8*
  %6 = bitcast %struct.point266* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo267(%struct.point267* noalias sret(%struct.point267) align 4 %0, %struct.point267* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point267* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point267* %0 to i8*
  %6 = bitcast %struct.point267* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo268(%struct.point268* noalias sret(%struct.point268) align 4 %0, %struct.point268* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point268* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point268* %0 to i8*
  %6 = bitcast %struct.point268* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo269([2 x i64] %0) #0 {
  %2 = alloca %struct.point269, align 4
  %3 = alloca %struct.point269, align 4
  %4 = bitcast %struct.point269* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point269* %2 to i8*
  %6 = bitcast %struct.point269* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point269* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo270(%struct.point270* noalias sret(%struct.point270) align 4 %0, %struct.point270* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point270* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point270* %0 to i8*
  %6 = bitcast %struct.point270* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo271(%struct.point271* noalias sret(%struct.point271) align 4 %0, %struct.point271* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point271* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point271* %0 to i8*
  %6 = bitcast %struct.point271* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo272(%struct.point272* noalias sret(%struct.point272) align 4 %0, %struct.point272* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point272* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point272* %0 to i8*
  %6 = bitcast %struct.point272* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo273(%struct.point273* noalias sret(%struct.point273) align 4 %0, %struct.point273* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point273* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point273* %0 to i8*
  %6 = bitcast %struct.point273* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo274([2 x i64] %0) #0 {
  %2 = alloca %struct.point274, align 4
  %3 = alloca %struct.point274, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point274* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point274* %2 to i8*
  %9 = bitcast %struct.point274* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point274* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo275([2 x i64] %0) #0 {
  %2 = alloca %struct.point275, align 4
  %3 = alloca %struct.point275, align 4
  %4 = bitcast %struct.point275* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point275* %2 to i8*
  %6 = bitcast %struct.point275* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point275* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo276([2 x i64] %0) #0 {
  %2 = alloca %struct.point276, align 4
  %3 = alloca %struct.point276, align 4
  %4 = bitcast %struct.point276* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point276* %2 to i8*
  %6 = bitcast %struct.point276* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point276* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo277([2 x i64] %0) #0 {
  %2 = alloca %struct.point277, align 4
  %3 = alloca %struct.point277, align 4
  %4 = bitcast %struct.point277* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point277* %2 to i8*
  %6 = bitcast %struct.point277* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point277* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo278(%struct.point278* noalias sret(%struct.point278) align 4 %0, %struct.point278* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point278* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point278* %0 to i8*
  %6 = bitcast %struct.point278* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo279(%struct.point279* noalias sret(%struct.point279) align 4 %0, %struct.point279* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point279* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point279* %0 to i8*
  %6 = bitcast %struct.point279* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo280([2 x i64] %0) #0 {
  %2 = alloca %struct.point280, align 4
  %3 = alloca %struct.point280, align 4
  %4 = bitcast %struct.point280* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point280* %2 to i8*
  %6 = bitcast %struct.point280* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point280* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo281([2 x i64] %0) #0 {
  %2 = alloca %struct.point281, align 4
  %3 = alloca %struct.point281, align 4
  %4 = bitcast %struct.point281* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point281* %2 to i8*
  %6 = bitcast %struct.point281* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point281* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo282([2 x i64] %0) #0 {
  %2 = alloca %struct.point282, align 4
  %3 = alloca %struct.point282, align 4
  %4 = bitcast %struct.point282* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point282* %2 to i8*
  %6 = bitcast %struct.point282* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point282* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo283(%struct.point283* noalias sret(%struct.point283) align 4 %0, %struct.point283* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point283* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point283* %0 to i8*
  %6 = bitcast %struct.point283* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo284(%struct.point284* noalias sret(%struct.point284) align 4 %0, %struct.point284* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point284* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point284* %0 to i8*
  %6 = bitcast %struct.point284* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo285([2 x i64] %0) #0 {
  %2 = alloca %struct.point285, align 4
  %3 = alloca %struct.point285, align 4
  %4 = bitcast %struct.point285* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point285* %2 to i8*
  %6 = bitcast %struct.point285* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point285* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo286(%struct.point286* noalias sret(%struct.point286) align 4 %0, %struct.point286* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point286* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point286* %0 to i8*
  %6 = bitcast %struct.point286* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo287(%struct.point287* noalias sret(%struct.point287) align 4 %0, %struct.point287* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point287* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point287* %0 to i8*
  %6 = bitcast %struct.point287* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo288(%struct.point288* noalias sret(%struct.point288) align 4 %0, %struct.point288* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point288* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point288* %0 to i8*
  %6 = bitcast %struct.point288* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo289(%struct.point289* noalias sret(%struct.point289) align 4 %0, %struct.point289* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point289* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point289* %0 to i8*
  %6 = bitcast %struct.point289* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo290([2 x i64] %0) #0 {
  %2 = alloca %struct.point290, align 4
  %3 = alloca %struct.point290, align 4
  %4 = bitcast %struct.point290* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point290* %2 to i8*
  %6 = bitcast %struct.point290* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point290* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo291(%struct.point291* noalias sret(%struct.point291) align 4 %0, %struct.point291* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point291* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point291* %0 to i8*
  %6 = bitcast %struct.point291* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo292(%struct.point292* noalias sret(%struct.point292) align 4 %0, %struct.point292* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point292* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point292* %0 to i8*
  %6 = bitcast %struct.point292* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo293(%struct.point293* noalias sret(%struct.point293) align 4 %0, %struct.point293* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point293* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point293* %0 to i8*
  %6 = bitcast %struct.point293* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo294(%struct.point294* noalias sret(%struct.point294) align 4 %0, %struct.point294* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point294* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point294* %0 to i8*
  %6 = bitcast %struct.point294* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo295(i64 %0) #0 {
  %2 = alloca %struct.point295, align 2
  %3 = alloca %struct.point295, align 2
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point295* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 4, i1 false)
  %8 = bitcast %struct.point295* %2 to i8*
  %9 = bitcast %struct.point295* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 4, i1 false)
  %10 = bitcast i64* %5 to i8*
  %11 = bitcast %struct.point295* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 4, i1 false)
  %12 = load i64, i64* %5, align 8
  ret i64 %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo296(i64 %0) #0 {
  %2 = alloca %struct.point296, align 2
  %3 = alloca %struct.point296, align 2
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point296* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 6, i1 false)
  %8 = bitcast %struct.point296* %2 to i8*
  %9 = bitcast %struct.point296* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 6, i1 false)
  %10 = bitcast i64* %5 to i8*
  %11 = bitcast %struct.point296* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 6, i1 false)
  %12 = load i64, i64* %5, align 8
  ret i64 %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo297(i64 %0) #0 {
  %2 = alloca %struct.point297, align 2
  %3 = alloca %struct.point297, align 2
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point297* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 6, i1 false)
  %8 = bitcast %struct.point297* %2 to i8*
  %9 = bitcast %struct.point297* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 6, i1 false)
  %10 = bitcast i64* %5 to i8*
  %11 = bitcast %struct.point297* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 6, i1 false)
  %12 = load i64, i64* %5, align 8
  ret i64 %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo298(i64 %0) #0 {
  %2 = alloca %struct.point298, align 2
  %3 = alloca %struct.point298, align 2
  %4 = bitcast %struct.point298* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point298* %2 to i8*
  %6 = bitcast %struct.point298* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point298* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo299([2 x i64] %0) #0 {
  %2 = alloca %struct.point299, align 4
  %3 = alloca %struct.point299, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point299* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point299* %2 to i8*
  %9 = bitcast %struct.point299* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point299* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo300([2 x i64] %0) #0 {
  %2 = alloca %struct.point300, align 4
  %3 = alloca %struct.point300, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point300* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point300* %2 to i8*
  %9 = bitcast %struct.point300* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point300* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo301(i64 %0) #0 {
  %2 = alloca %struct.point301, align 2
  %3 = alloca %struct.point301, align 2
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point301* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 6, i1 false)
  %8 = bitcast %struct.point301* %2 to i8*
  %9 = bitcast %struct.point301* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 6, i1 false)
  %10 = bitcast i64* %5 to i8*
  %11 = bitcast %struct.point301* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 6, i1 false)
  %12 = load i64, i64* %5, align 8
  ret i64 %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo302(i64 %0) #0 {
  %2 = alloca %struct.point302, align 2
  %3 = alloca %struct.point302, align 2
  %4 = bitcast %struct.point302* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point302* %2 to i8*
  %6 = bitcast %struct.point302* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point302* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo303(i64 %0) #0 {
  %2 = alloca %struct.point303, align 2
  %3 = alloca %struct.point303, align 2
  %4 = bitcast %struct.point303* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point303* %2 to i8*
  %6 = bitcast %struct.point303* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point303* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo304([2 x i64] %0) #0 {
  %2 = alloca %struct.point304, align 4
  %3 = alloca %struct.point304, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point304* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point304* %2 to i8*
  %9 = bitcast %struct.point304* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point304* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo305([2 x i64] %0) #0 {
  %2 = alloca %struct.point305, align 4
  %3 = alloca %struct.point305, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point305* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point305* %2 to i8*
  %9 = bitcast %struct.point305* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point305* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo306(i64 %0) #0 {
  %2 = alloca %struct.point306, align 4
  %3 = alloca %struct.point306, align 4
  %4 = bitcast %struct.point306* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point306* %2 to i8*
  %6 = bitcast %struct.point306* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point306* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo307([2 x i64] %0) #0 {
  %2 = alloca %struct.point307, align 4
  %3 = alloca %struct.point307, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point307* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point307* %2 to i8*
  %9 = bitcast %struct.point307* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point307* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo308([2 x i64] %0) #0 {
  %2 = alloca %struct.point308, align 4
  %3 = alloca %struct.point308, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point308* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point308* %2 to i8*
  %9 = bitcast %struct.point308* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point308* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo309([2 x i64] %0) #0 {
  %2 = alloca %struct.point309, align 4
  %3 = alloca %struct.point309, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point309* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point309* %2 to i8*
  %9 = bitcast %struct.point309* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point309* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo310([2 x i64] %0) #0 {
  %2 = alloca %struct.point310, align 4
  %3 = alloca %struct.point310, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point310* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point310* %2 to i8*
  %9 = bitcast %struct.point310* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point310* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo311(i64 %0) #0 {
  %2 = alloca %struct.point311, align 4
  %3 = alloca %struct.point311, align 4
  %4 = bitcast %struct.point311* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point311* %2 to i8*
  %6 = bitcast %struct.point311* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point311* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo312([2 x i64] %0) #0 {
  %2 = alloca %struct.point312, align 4
  %3 = alloca %struct.point312, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point312* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point312* %2 to i8*
  %9 = bitcast %struct.point312* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point312* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo313([2 x i64] %0) #0 {
  %2 = alloca %struct.point313, align 4
  %3 = alloca %struct.point313, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point313* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point313* %2 to i8*
  %9 = bitcast %struct.point313* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point313* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo314([2 x i64] %0) #0 {
  %2 = alloca %struct.point314, align 4
  %3 = alloca %struct.point314, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point314* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point314* %2 to i8*
  %9 = bitcast %struct.point314* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point314* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo315([2 x i64] %0) #0 {
  %2 = alloca %struct.point315, align 4
  %3 = alloca %struct.point315, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point315* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point315* %2 to i8*
  %9 = bitcast %struct.point315* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point315* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo316(i64 %0) #0 {
  %2 = alloca %struct.point316, align 2
  %3 = alloca %struct.point316, align 2
  %4 = alloca i64, align 8
  %5 = alloca i64, align 8
  store i64 %0, i64* %4, align 8
  %6 = bitcast %struct.point316* %3 to i8*
  %7 = bitcast i64* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 6, i1 false)
  %8 = bitcast %struct.point316* %2 to i8*
  %9 = bitcast %struct.point316* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 6, i1 false)
  %10 = bitcast i64* %5 to i8*
  %11 = bitcast %struct.point316* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 6, i1 false)
  %12 = load i64, i64* %5, align 8
  ret i64 %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo317(i64 %0) #0 {
  %2 = alloca %struct.point317, align 2
  %3 = alloca %struct.point317, align 2
  %4 = bitcast %struct.point317* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point317* %2 to i8*
  %6 = bitcast %struct.point317* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point317* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo318(i64 %0) #0 {
  %2 = alloca %struct.point318, align 2
  %3 = alloca %struct.point318, align 2
  %4 = bitcast %struct.point318* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point318* %2 to i8*
  %6 = bitcast %struct.point318* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point318* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo319([2 x i64] %0) #0 {
  %2 = alloca %struct.point319, align 2
  %3 = alloca %struct.point319, align 2
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point319* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 10, i1 false)
  %8 = bitcast %struct.point319* %2 to i8*
  %9 = bitcast %struct.point319* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 10, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point319* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 10, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo320([2 x i64] %0) #0 {
  %2 = alloca %struct.point320, align 4
  %3 = alloca %struct.point320, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point320* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point320* %2 to i8*
  %9 = bitcast %struct.point320* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point320* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo321([2 x i64] %0) #0 {
  %2 = alloca %struct.point321, align 4
  %3 = alloca %struct.point321, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point321* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point321* %2 to i8*
  %9 = bitcast %struct.point321* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point321* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo322(i64 %0) #0 {
  %2 = alloca %struct.point322, align 2
  %3 = alloca %struct.point322, align 2
  %4 = bitcast %struct.point322* %3 to i64*
  store i64 %0, i64* %4, align 2
  %5 = bitcast %struct.point322* %2 to i8*
  %6 = bitcast %struct.point322* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %5, i8* align 2 %6, i64 8, i1 false)
  %7 = bitcast %struct.point322* %2 to i64*
  %8 = load i64, i64* %7, align 2
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo323([2 x i64] %0) #0 {
  %2 = alloca %struct.point323, align 2
  %3 = alloca %struct.point323, align 2
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point323* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 10, i1 false)
  %8 = bitcast %struct.point323* %2 to i8*
  %9 = bitcast %struct.point323* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 10, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point323* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 10, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo324([2 x i64] %0) #0 {
  %2 = alloca %struct.point324, align 2
  %3 = alloca %struct.point324, align 2
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point324* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %6, i8* align 8 %7, i64 10, i1 false)
  %8 = bitcast %struct.point324* %2 to i8*
  %9 = bitcast %struct.point324* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 2 %8, i8* align 2 %9, i64 10, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point324* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 2 %11, i64 10, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo325([2 x i64] %0) #0 {
  %2 = alloca %struct.point325, align 4
  %3 = alloca %struct.point325, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point325* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point325* %2 to i8*
  %9 = bitcast %struct.point325* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point325* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo326([2 x i64] %0) #0 {
  %2 = alloca %struct.point326, align 4
  %3 = alloca %struct.point326, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point326* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point326* %2 to i8*
  %9 = bitcast %struct.point326* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point326* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo327([2 x i64] %0) #0 {
  %2 = alloca %struct.point327, align 4
  %3 = alloca %struct.point327, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point327* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point327* %2 to i8*
  %9 = bitcast %struct.point327* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point327* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo328([2 x i64] %0) #0 {
  %2 = alloca %struct.point328, align 4
  %3 = alloca %struct.point328, align 4
  %4 = bitcast %struct.point328* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point328* %2 to i8*
  %6 = bitcast %struct.point328* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point328* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo329([2 x i64] %0) #0 {
  %2 = alloca %struct.point329, align 4
  %3 = alloca %struct.point329, align 4
  %4 = bitcast %struct.point329* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point329* %2 to i8*
  %6 = bitcast %struct.point329* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point329* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo330([2 x i64] %0) #0 {
  %2 = alloca %struct.point330, align 4
  %3 = alloca %struct.point330, align 4
  %4 = bitcast %struct.point330* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point330* %2 to i8*
  %6 = bitcast %struct.point330* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point330* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo331([2 x i64] %0) #0 {
  %2 = alloca %struct.point331, align 4
  %3 = alloca %struct.point331, align 4
  %4 = bitcast %struct.point331* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point331* %2 to i8*
  %6 = bitcast %struct.point331* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point331* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo332([2 x i64] %0) #0 {
  %2 = alloca %struct.point332, align 4
  %3 = alloca %struct.point332, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point332* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point332* %2 to i8*
  %9 = bitcast %struct.point332* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point332* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo333([2 x i64] %0) #0 {
  %2 = alloca %struct.point333, align 4
  %3 = alloca %struct.point333, align 4
  %4 = bitcast %struct.point333* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point333* %2 to i8*
  %6 = bitcast %struct.point333* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point333* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo334([2 x i64] %0) #0 {
  %2 = alloca %struct.point334, align 4
  %3 = alloca %struct.point334, align 4
  %4 = bitcast %struct.point334* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point334* %2 to i8*
  %6 = bitcast %struct.point334* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point334* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo335([2 x i64] %0) #0 {
  %2 = alloca %struct.point335, align 4
  %3 = alloca %struct.point335, align 4
  %4 = bitcast %struct.point335* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point335* %2 to i8*
  %6 = bitcast %struct.point335* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point335* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo336([2 x i64] %0) #0 {
  %2 = alloca %struct.point336, align 4
  %3 = alloca %struct.point336, align 4
  %4 = bitcast %struct.point336* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point336* %2 to i8*
  %6 = bitcast %struct.point336* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point336* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo337(i64 %0) #0 {
  %2 = alloca %struct.point337, align 4
  %3 = alloca %struct.point337, align 4
  %4 = bitcast %struct.point337* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point337* %2 to i8*
  %6 = bitcast %struct.point337* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point337* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo338([2 x i64] %0) #0 {
  %2 = alloca %struct.point338, align 4
  %3 = alloca %struct.point338, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point338* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point338* %2 to i8*
  %9 = bitcast %struct.point338* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point338* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo339([2 x i64] %0) #0 {
  %2 = alloca %struct.point339, align 4
  %3 = alloca %struct.point339, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point339* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point339* %2 to i8*
  %9 = bitcast %struct.point339* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point339* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo340([2 x i64] %0) #0 {
  %2 = alloca %struct.point340, align 4
  %3 = alloca %struct.point340, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point340* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point340* %2 to i8*
  %9 = bitcast %struct.point340* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point340* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo341([2 x i64] %0) #0 {
  %2 = alloca %struct.point341, align 4
  %3 = alloca %struct.point341, align 4
  %4 = bitcast %struct.point341* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point341* %2 to i8*
  %6 = bitcast %struct.point341* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point341* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo342([2 x i64] %0) #0 {
  %2 = alloca %struct.point342, align 4
  %3 = alloca %struct.point342, align 4
  %4 = bitcast %struct.point342* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point342* %2 to i8*
  %6 = bitcast %struct.point342* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point342* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo343([2 x i64] %0) #0 {
  %2 = alloca %struct.point343, align 4
  %3 = alloca %struct.point343, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point343* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point343* %2 to i8*
  %9 = bitcast %struct.point343* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point343* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo344([2 x i64] %0) #0 {
  %2 = alloca %struct.point344, align 4
  %3 = alloca %struct.point344, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point344* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point344* %2 to i8*
  %9 = bitcast %struct.point344* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point344* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo345([2 x i64] %0) #0 {
  %2 = alloca %struct.point345, align 4
  %3 = alloca %struct.point345, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point345* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point345* %2 to i8*
  %9 = bitcast %struct.point345* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point345* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo346([2 x i64] %0) #0 {
  %2 = alloca %struct.point346, align 4
  %3 = alloca %struct.point346, align 4
  %4 = bitcast %struct.point346* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point346* %2 to i8*
  %6 = bitcast %struct.point346* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point346* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo347([2 x i64] %0) #0 {
  %2 = alloca %struct.point347, align 4
  %3 = alloca %struct.point347, align 4
  %4 = bitcast %struct.point347* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point347* %2 to i8*
  %6 = bitcast %struct.point347* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point347* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo348([2 x i64] %0) #0 {
  %2 = alloca %struct.point348, align 4
  %3 = alloca %struct.point348, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point348* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point348* %2 to i8*
  %9 = bitcast %struct.point348* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point348* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo349([2 x i64] %0) #0 {
  %2 = alloca %struct.point349, align 4
  %3 = alloca %struct.point349, align 4
  %4 = bitcast %struct.point349* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point349* %2 to i8*
  %6 = bitcast %struct.point349* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point349* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo350([2 x i64] %0) #0 {
  %2 = alloca %struct.point350, align 4
  %3 = alloca %struct.point350, align 4
  %4 = bitcast %struct.point350* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point350* %2 to i8*
  %6 = bitcast %struct.point350* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point350* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo351([2 x i64] %0) #0 {
  %2 = alloca %struct.point351, align 4
  %3 = alloca %struct.point351, align 4
  %4 = bitcast %struct.point351* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point351* %2 to i8*
  %6 = bitcast %struct.point351* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point351* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo352([2 x i64] %0) #0 {
  %2 = alloca %struct.point352, align 4
  %3 = alloca %struct.point352, align 4
  %4 = bitcast %struct.point352* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point352* %2 to i8*
  %6 = bitcast %struct.point352* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point352* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo353([2 x i64] %0) #0 {
  %2 = alloca %struct.point353, align 4
  %3 = alloca %struct.point353, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point353* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point353* %2 to i8*
  %9 = bitcast %struct.point353* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point353* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo354([2 x i64] %0) #0 {
  %2 = alloca %struct.point354, align 4
  %3 = alloca %struct.point354, align 4
  %4 = bitcast %struct.point354* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point354* %2 to i8*
  %6 = bitcast %struct.point354* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point354* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo355([2 x i64] %0) #0 {
  %2 = alloca %struct.point355, align 4
  %3 = alloca %struct.point355, align 4
  %4 = bitcast %struct.point355* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point355* %2 to i8*
  %6 = bitcast %struct.point355* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point355* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo356([2 x i64] %0) #0 {
  %2 = alloca %struct.point356, align 4
  %3 = alloca %struct.point356, align 4
  %4 = bitcast %struct.point356* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point356* %2 to i8*
  %6 = bitcast %struct.point356* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point356* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo357([2 x i64] %0) #0 {
  %2 = alloca %struct.point357, align 4
  %3 = alloca %struct.point357, align 4
  %4 = bitcast %struct.point357* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point357* %2 to i8*
  %6 = bitcast %struct.point357* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point357* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo358(i64 %0) #0 {
  %2 = alloca %struct.point358, align 4
  %3 = alloca %struct.point358, align 4
  %4 = bitcast %struct.point358* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point358* %2 to i8*
  %6 = bitcast %struct.point358* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point358* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo359([2 x i64] %0) #0 {
  %2 = alloca %struct.point359, align 4
  %3 = alloca %struct.point359, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point359* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point359* %2 to i8*
  %9 = bitcast %struct.point359* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point359* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo360([2 x i64] %0) #0 {
  %2 = alloca %struct.point360, align 4
  %3 = alloca %struct.point360, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point360* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point360* %2 to i8*
  %9 = bitcast %struct.point360* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point360* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo361([2 x i64] %0) #0 {
  %2 = alloca %struct.point361, align 4
  %3 = alloca %struct.point361, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point361* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point361* %2 to i8*
  %9 = bitcast %struct.point361* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point361* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo362([2 x i64] %0) #0 {
  %2 = alloca %struct.point362, align 4
  %3 = alloca %struct.point362, align 4
  %4 = bitcast %struct.point362* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point362* %2 to i8*
  %6 = bitcast %struct.point362* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point362* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo363([2 x i64] %0) #0 {
  %2 = alloca %struct.point363, align 4
  %3 = alloca %struct.point363, align 4
  %4 = bitcast %struct.point363* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point363* %2 to i8*
  %6 = bitcast %struct.point363* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point363* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo364([2 x i64] %0) #0 {
  %2 = alloca %struct.point364, align 4
  %3 = alloca %struct.point364, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point364* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point364* %2 to i8*
  %9 = bitcast %struct.point364* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point364* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo365([2 x i64] %0) #0 {
  %2 = alloca %struct.point365, align 4
  %3 = alloca %struct.point365, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point365* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point365* %2 to i8*
  %9 = bitcast %struct.point365* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point365* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo366([2 x i64] %0) #0 {
  %2 = alloca %struct.point366, align 4
  %3 = alloca %struct.point366, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point366* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point366* %2 to i8*
  %9 = bitcast %struct.point366* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point366* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo367([2 x i64] %0) #0 {
  %2 = alloca %struct.point367, align 4
  %3 = alloca %struct.point367, align 4
  %4 = bitcast %struct.point367* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point367* %2 to i8*
  %6 = bitcast %struct.point367* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point367* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo368([2 x i64] %0) #0 {
  %2 = alloca %struct.point368, align 4
  %3 = alloca %struct.point368, align 4
  %4 = bitcast %struct.point368* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point368* %2 to i8*
  %6 = bitcast %struct.point368* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point368* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo369([2 x i64] %0) #0 {
  %2 = alloca %struct.point369, align 4
  %3 = alloca %struct.point369, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point369* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point369* %2 to i8*
  %9 = bitcast %struct.point369* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point369* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo370([2 x i64] %0) #0 {
  %2 = alloca %struct.point370, align 4
  %3 = alloca %struct.point370, align 4
  %4 = bitcast %struct.point370* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point370* %2 to i8*
  %6 = bitcast %struct.point370* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point370* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo371([2 x i64] %0) #0 {
  %2 = alloca %struct.point371, align 4
  %3 = alloca %struct.point371, align 4
  %4 = bitcast %struct.point371* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point371* %2 to i8*
  %6 = bitcast %struct.point371* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point371* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo372([2 x i64] %0) #0 {
  %2 = alloca %struct.point372, align 4
  %3 = alloca %struct.point372, align 4
  %4 = bitcast %struct.point372* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point372* %2 to i8*
  %6 = bitcast %struct.point372* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point372* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo373([2 x i64] %0) #0 {
  %2 = alloca %struct.point373, align 4
  %3 = alloca %struct.point373, align 4
  %4 = bitcast %struct.point373* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point373* %2 to i8*
  %6 = bitcast %struct.point373* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point373* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo374([2 x i64] %0) #0 {
  %2 = alloca %struct.point374, align 4
  %3 = alloca %struct.point374, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point374* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point374* %2 to i8*
  %9 = bitcast %struct.point374* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point374* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo375([2 x i64] %0) #0 {
  %2 = alloca %struct.point375, align 4
  %3 = alloca %struct.point375, align 4
  %4 = bitcast %struct.point375* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point375* %2 to i8*
  %6 = bitcast %struct.point375* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point375* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo376([2 x i64] %0) #0 {
  %2 = alloca %struct.point376, align 4
  %3 = alloca %struct.point376, align 4
  %4 = bitcast %struct.point376* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point376* %2 to i8*
  %6 = bitcast %struct.point376* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point376* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo377([2 x i64] %0) #0 {
  %2 = alloca %struct.point377, align 4
  %3 = alloca %struct.point377, align 4
  %4 = bitcast %struct.point377* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point377* %2 to i8*
  %6 = bitcast %struct.point377* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point377* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo378([2 x i64] %0) #0 {
  %2 = alloca %struct.point378, align 4
  %3 = alloca %struct.point378, align 4
  %4 = bitcast %struct.point378* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point378* %2 to i8*
  %6 = bitcast %struct.point378* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point378* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo379([2 x i64] %0) #0 {
  %2 = alloca %struct.point379, align 4
  %3 = alloca %struct.point379, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point379* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point379* %2 to i8*
  %9 = bitcast %struct.point379* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point379* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo380([2 x i64] %0) #0 {
  %2 = alloca %struct.point380, align 4
  %3 = alloca %struct.point380, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point380* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point380* %2 to i8*
  %9 = bitcast %struct.point380* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point380* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo381([2 x i64] %0) #0 {
  %2 = alloca %struct.point381, align 4
  %3 = alloca %struct.point381, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point381* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point381* %2 to i8*
  %9 = bitcast %struct.point381* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point381* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo382([2 x i64] %0) #0 {
  %2 = alloca %struct.point382, align 4
  %3 = alloca %struct.point382, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point382* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point382* %2 to i8*
  %9 = bitcast %struct.point382* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point382* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo383([2 x i64] %0) #0 {
  %2 = alloca %struct.point383, align 4
  %3 = alloca %struct.point383, align 4
  %4 = bitcast %struct.point383* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point383* %2 to i8*
  %6 = bitcast %struct.point383* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point383* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo384([2 x i64] %0) #0 {
  %2 = alloca %struct.point384, align 4
  %3 = alloca %struct.point384, align 4
  %4 = bitcast %struct.point384* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point384* %2 to i8*
  %6 = bitcast %struct.point384* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point384* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo385([2 x i64] %0) #0 {
  %2 = alloca %struct.point385, align 4
  %3 = alloca %struct.point385, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point385* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point385* %2 to i8*
  %9 = bitcast %struct.point385* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point385* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo386([2 x i64] %0) #0 {
  %2 = alloca %struct.point386, align 4
  %3 = alloca %struct.point386, align 4
  %4 = bitcast %struct.point386* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point386* %2 to i8*
  %6 = bitcast %struct.point386* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point386* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo387([2 x i64] %0) #0 {
  %2 = alloca %struct.point387, align 4
  %3 = alloca %struct.point387, align 4
  %4 = bitcast %struct.point387* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point387* %2 to i8*
  %6 = bitcast %struct.point387* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point387* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo388([2 x i64] %0) #0 {
  %2 = alloca %struct.point388, align 4
  %3 = alloca %struct.point388, align 4
  %4 = bitcast %struct.point388* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point388* %2 to i8*
  %6 = bitcast %struct.point388* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point388* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo389([2 x i64] %0) #0 {
  %2 = alloca %struct.point389, align 4
  %3 = alloca %struct.point389, align 4
  %4 = bitcast %struct.point389* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point389* %2 to i8*
  %6 = bitcast %struct.point389* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point389* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo390([2 x i64] %0) #0 {
  %2 = alloca %struct.point390, align 4
  %3 = alloca %struct.point390, align 4
  %4 = bitcast %struct.point390* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point390* %2 to i8*
  %6 = bitcast %struct.point390* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point390* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo391(%struct.point391* noalias sret(%struct.point391) align 4 %0, %struct.point391* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point391* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point391* %0 to i8*
  %6 = bitcast %struct.point391* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo392(%struct.point392* noalias sret(%struct.point392) align 4 %0, %struct.point392* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point392* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point392* %0 to i8*
  %6 = bitcast %struct.point392* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo393(%struct.point393* noalias sret(%struct.point393) align 4 %0, %struct.point393* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point393* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point393* %0 to i8*
  %6 = bitcast %struct.point393* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo394(%struct.point394* noalias sret(%struct.point394) align 4 %0, %struct.point394* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point394* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point394* %0 to i8*
  %6 = bitcast %struct.point394* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo395([2 x i64] %0) #0 {
  %2 = alloca %struct.point395, align 4
  %3 = alloca %struct.point395, align 4
  %4 = bitcast %struct.point395* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point395* %2 to i8*
  %6 = bitcast %struct.point395* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point395* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo396(%struct.point396* noalias sret(%struct.point396) align 4 %0, %struct.point396* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point396* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point396* %0 to i8*
  %6 = bitcast %struct.point396* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo397(%struct.point397* noalias sret(%struct.point397) align 4 %0, %struct.point397* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point397* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point397* %0 to i8*
  %6 = bitcast %struct.point397* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo398(%struct.point398* noalias sret(%struct.point398) align 4 %0, %struct.point398* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point398* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point398* %0 to i8*
  %6 = bitcast %struct.point398* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo399(%struct.point399* noalias sret(%struct.point399) align 4 %0, %struct.point399* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point399* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point399* %0 to i8*
  %6 = bitcast %struct.point399* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo400([2 x i64] %0) #0 {
  %2 = alloca %struct.point400, align 4
  %3 = alloca %struct.point400, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point400* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point400* %2 to i8*
  %9 = bitcast %struct.point400* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point400* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo401([2 x i64] %0) #0 {
  %2 = alloca %struct.point401, align 4
  %3 = alloca %struct.point401, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point401* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point401* %2 to i8*
  %9 = bitcast %struct.point401* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point401* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo402([2 x i64] %0) #0 {
  %2 = alloca %struct.point402, align 4
  %3 = alloca %struct.point402, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point402* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point402* %2 to i8*
  %9 = bitcast %struct.point402* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point402* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo403([2 x i64] %0) #0 {
  %2 = alloca %struct.point403, align 4
  %3 = alloca %struct.point403, align 4
  %4 = bitcast %struct.point403* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point403* %2 to i8*
  %6 = bitcast %struct.point403* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point403* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo404([2 x i64] %0) #0 {
  %2 = alloca %struct.point404, align 4
  %3 = alloca %struct.point404, align 4
  %4 = bitcast %struct.point404* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point404* %2 to i8*
  %6 = bitcast %struct.point404* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point404* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo405([2 x i64] %0) #0 {
  %2 = alloca %struct.point405, align 4
  %3 = alloca %struct.point405, align 4
  %4 = bitcast %struct.point405* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point405* %2 to i8*
  %6 = bitcast %struct.point405* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point405* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo406([2 x i64] %0) #0 {
  %2 = alloca %struct.point406, align 4
  %3 = alloca %struct.point406, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point406* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point406* %2 to i8*
  %9 = bitcast %struct.point406* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point406* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo407([2 x i64] %0) #0 {
  %2 = alloca %struct.point407, align 4
  %3 = alloca %struct.point407, align 4
  %4 = bitcast %struct.point407* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point407* %2 to i8*
  %6 = bitcast %struct.point407* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point407* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo408([2 x i64] %0) #0 {
  %2 = alloca %struct.point408, align 4
  %3 = alloca %struct.point408, align 4
  %4 = bitcast %struct.point408* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point408* %2 to i8*
  %6 = bitcast %struct.point408* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point408* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo409([2 x i64] %0) #0 {
  %2 = alloca %struct.point409, align 4
  %3 = alloca %struct.point409, align 4
  %4 = bitcast %struct.point409* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point409* %2 to i8*
  %6 = bitcast %struct.point409* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point409* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo410([2 x i64] %0) #0 {
  %2 = alloca %struct.point410, align 4
  %3 = alloca %struct.point410, align 4
  %4 = bitcast %struct.point410* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point410* %2 to i8*
  %6 = bitcast %struct.point410* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point410* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo411([2 x i64] %0) #0 {
  %2 = alloca %struct.point411, align 4
  %3 = alloca %struct.point411, align 4
  %4 = bitcast %struct.point411* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point411* %2 to i8*
  %6 = bitcast %struct.point411* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point411* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo412(%struct.point412* noalias sret(%struct.point412) align 4 %0, %struct.point412* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point412* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point412* %0 to i8*
  %6 = bitcast %struct.point412* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo413(%struct.point413* noalias sret(%struct.point413) align 4 %0, %struct.point413* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point413* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point413* %0 to i8*
  %6 = bitcast %struct.point413* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo414(%struct.point414* noalias sret(%struct.point414) align 4 %0, %struct.point414* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point414* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point414* %0 to i8*
  %6 = bitcast %struct.point414* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo415(%struct.point415* noalias sret(%struct.point415) align 4 %0, %struct.point415* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point415* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point415* %0 to i8*
  %6 = bitcast %struct.point415* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo416([2 x i64] %0) #0 {
  %2 = alloca %struct.point416, align 4
  %3 = alloca %struct.point416, align 4
  %4 = bitcast %struct.point416* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point416* %2 to i8*
  %6 = bitcast %struct.point416* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point416* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo417(%struct.point417* noalias sret(%struct.point417) align 4 %0, %struct.point417* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point417* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point417* %0 to i8*
  %6 = bitcast %struct.point417* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo418(%struct.point418* noalias sret(%struct.point418) align 4 %0, %struct.point418* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point418* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point418* %0 to i8*
  %6 = bitcast %struct.point418* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo419(%struct.point419* noalias sret(%struct.point419) align 4 %0, %struct.point419* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point419* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point419* %0 to i8*
  %6 = bitcast %struct.point419* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo420(%struct.point420* noalias sret(%struct.point420) align 4 %0, %struct.point420* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point420* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point420* %0 to i8*
  %6 = bitcast %struct.point420* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo421([2 x i64] %0) #0 {
  %2 = alloca %struct.point421, align 4
  %3 = alloca %struct.point421, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point421* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point421* %2 to i8*
  %9 = bitcast %struct.point421* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point421* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo422([2 x i64] %0) #0 {
  %2 = alloca %struct.point422, align 4
  %3 = alloca %struct.point422, align 4
  %4 = bitcast %struct.point422* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point422* %2 to i8*
  %6 = bitcast %struct.point422* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point422* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo423([2 x i64] %0) #0 {
  %2 = alloca %struct.point423, align 4
  %3 = alloca %struct.point423, align 4
  %4 = bitcast %struct.point423* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point423* %2 to i8*
  %6 = bitcast %struct.point423* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point423* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo424([2 x i64] %0) #0 {
  %2 = alloca %struct.point424, align 4
  %3 = alloca %struct.point424, align 4
  %4 = bitcast %struct.point424* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point424* %2 to i8*
  %6 = bitcast %struct.point424* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point424* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo425(%struct.point425* noalias sret(%struct.point425) align 4 %0, %struct.point425* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point425* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point425* %0 to i8*
  %6 = bitcast %struct.point425* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo426(%struct.point426* noalias sret(%struct.point426) align 4 %0, %struct.point426* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point426* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point426* %0 to i8*
  %6 = bitcast %struct.point426* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo427([2 x i64] %0) #0 {
  %2 = alloca %struct.point427, align 4
  %3 = alloca %struct.point427, align 4
  %4 = bitcast %struct.point427* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point427* %2 to i8*
  %6 = bitcast %struct.point427* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point427* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo428([2 x i64] %0) #0 {
  %2 = alloca %struct.point428, align 4
  %3 = alloca %struct.point428, align 4
  %4 = bitcast %struct.point428* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point428* %2 to i8*
  %6 = bitcast %struct.point428* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point428* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo429([2 x i64] %0) #0 {
  %2 = alloca %struct.point429, align 4
  %3 = alloca %struct.point429, align 4
  %4 = bitcast %struct.point429* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point429* %2 to i8*
  %6 = bitcast %struct.point429* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point429* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo430(%struct.point430* noalias sret(%struct.point430) align 4 %0, %struct.point430* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point430* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point430* %0 to i8*
  %6 = bitcast %struct.point430* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo431(%struct.point431* noalias sret(%struct.point431) align 4 %0, %struct.point431* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point431* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point431* %0 to i8*
  %6 = bitcast %struct.point431* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo432([2 x i64] %0) #0 {
  %2 = alloca %struct.point432, align 4
  %3 = alloca %struct.point432, align 4
  %4 = bitcast %struct.point432* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point432* %2 to i8*
  %6 = bitcast %struct.point432* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point432* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo433(%struct.point433* noalias sret(%struct.point433) align 4 %0, %struct.point433* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point433* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point433* %0 to i8*
  %6 = bitcast %struct.point433* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo434(%struct.point434* noalias sret(%struct.point434) align 4 %0, %struct.point434* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point434* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point434* %0 to i8*
  %6 = bitcast %struct.point434* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo435(%struct.point435* noalias sret(%struct.point435) align 4 %0, %struct.point435* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point435* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point435* %0 to i8*
  %6 = bitcast %struct.point435* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo436(%struct.point436* noalias sret(%struct.point436) align 4 %0, %struct.point436* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point436* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point436* %0 to i8*
  %6 = bitcast %struct.point436* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo437([2 x i64] %0) #0 {
  %2 = alloca %struct.point437, align 4
  %3 = alloca %struct.point437, align 4
  %4 = bitcast %struct.point437* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point437* %2 to i8*
  %6 = bitcast %struct.point437* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point437* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo438(%struct.point438* noalias sret(%struct.point438) align 4 %0, %struct.point438* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point438* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point438* %0 to i8*
  %6 = bitcast %struct.point438* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo439(%struct.point439* noalias sret(%struct.point439) align 4 %0, %struct.point439* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point439* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point439* %0 to i8*
  %6 = bitcast %struct.point439* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo440(%struct.point440* noalias sret(%struct.point440) align 4 %0, %struct.point440* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point440* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point440* %0 to i8*
  %6 = bitcast %struct.point440* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo441(%struct.point441* noalias sret(%struct.point441) align 4 %0, %struct.point441* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point441* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point441* %0 to i8*
  %6 = bitcast %struct.point441* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo442([2 x i64] %0) #0 {
  %2 = alloca %struct.point442, align 4
  %3 = alloca %struct.point442, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point442* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point442* %2 to i8*
  %9 = bitcast %struct.point442* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point442* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo443([2 x i64] %0) #0 {
  %2 = alloca %struct.point443, align 4
  %3 = alloca %struct.point443, align 4
  %4 = bitcast %struct.point443* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point443* %2 to i8*
  %6 = bitcast %struct.point443* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point443* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo444([2 x i64] %0) #0 {
  %2 = alloca %struct.point444, align 4
  %3 = alloca %struct.point444, align 4
  %4 = bitcast %struct.point444* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point444* %2 to i8*
  %6 = bitcast %struct.point444* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point444* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo445([2 x i64] %0) #0 {
  %2 = alloca %struct.point445, align 4
  %3 = alloca %struct.point445, align 4
  %4 = bitcast %struct.point445* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point445* %2 to i8*
  %6 = bitcast %struct.point445* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point445* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo446(%struct.point446* noalias sret(%struct.point446) align 4 %0, %struct.point446* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point446* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point446* %0 to i8*
  %6 = bitcast %struct.point446* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo447(%struct.point447* noalias sret(%struct.point447) align 4 %0, %struct.point447* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point447* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point447* %0 to i8*
  %6 = bitcast %struct.point447* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo448([2 x i64] %0) #0 {
  %2 = alloca %struct.point448, align 4
  %3 = alloca %struct.point448, align 4
  %4 = bitcast %struct.point448* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point448* %2 to i8*
  %6 = bitcast %struct.point448* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point448* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo449([2 x i64] %0) #0 {
  %2 = alloca %struct.point449, align 4
  %3 = alloca %struct.point449, align 4
  %4 = bitcast %struct.point449* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point449* %2 to i8*
  %6 = bitcast %struct.point449* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point449* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo450([2 x i64] %0) #0 {
  %2 = alloca %struct.point450, align 4
  %3 = alloca %struct.point450, align 4
  %4 = bitcast %struct.point450* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point450* %2 to i8*
  %6 = bitcast %struct.point450* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point450* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo451(%struct.point451* noalias sret(%struct.point451) align 4 %0, %struct.point451* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point451* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point451* %0 to i8*
  %6 = bitcast %struct.point451* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo452(%struct.point452* noalias sret(%struct.point452) align 4 %0, %struct.point452* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point452* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point452* %0 to i8*
  %6 = bitcast %struct.point452* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo453([2 x i64] %0) #0 {
  %2 = alloca %struct.point453, align 4
  %3 = alloca %struct.point453, align 4
  %4 = bitcast %struct.point453* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point453* %2 to i8*
  %6 = bitcast %struct.point453* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point453* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo454(%struct.point454* noalias sret(%struct.point454) align 4 %0, %struct.point454* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point454* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point454* %0 to i8*
  %6 = bitcast %struct.point454* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo455(%struct.point455* noalias sret(%struct.point455) align 4 %0, %struct.point455* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point455* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point455* %0 to i8*
  %6 = bitcast %struct.point455* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo456(%struct.point456* noalias sret(%struct.point456) align 4 %0, %struct.point456* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point456* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point456* %0 to i8*
  %6 = bitcast %struct.point456* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo457(%struct.point457* noalias sret(%struct.point457) align 4 %0, %struct.point457* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point457* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point457* %0 to i8*
  %6 = bitcast %struct.point457* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo458([2 x i64] %0) #0 {
  %2 = alloca %struct.point458, align 4
  %3 = alloca %struct.point458, align 4
  %4 = bitcast %struct.point458* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point458* %2 to i8*
  %6 = bitcast %struct.point458* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point458* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo459(%struct.point459* noalias sret(%struct.point459) align 4 %0, %struct.point459* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point459* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point459* %0 to i8*
  %6 = bitcast %struct.point459* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo460(%struct.point460* noalias sret(%struct.point460) align 4 %0, %struct.point460* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point460* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point460* %0 to i8*
  %6 = bitcast %struct.point460* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo461(%struct.point461* noalias sret(%struct.point461) align 4 %0, %struct.point461* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point461* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point461* %0 to i8*
  %6 = bitcast %struct.point461* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo462(%struct.point462* noalias sret(%struct.point462) align 4 %0, %struct.point462* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point462* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point462* %0 to i8*
  %6 = bitcast %struct.point462* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo463([2 x i64] %0) #0 {
  %2 = alloca %struct.point463, align 4
  %3 = alloca %struct.point463, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point463* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point463* %2 to i8*
  %9 = bitcast %struct.point463* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point463* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo464([2 x i64] %0) #0 {
  %2 = alloca %struct.point464, align 4
  %3 = alloca %struct.point464, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point464* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point464* %2 to i8*
  %9 = bitcast %struct.point464* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point464* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo465([2 x i64] %0) #0 {
  %2 = alloca %struct.point465, align 4
  %3 = alloca %struct.point465, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point465* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point465* %2 to i8*
  %9 = bitcast %struct.point465* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point465* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo466([2 x i64] %0) #0 {
  %2 = alloca %struct.point466, align 4
  %3 = alloca %struct.point466, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point466* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point466* %2 to i8*
  %9 = bitcast %struct.point466* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point466* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo467([2 x i64] %0) #0 {
  %2 = alloca %struct.point467, align 4
  %3 = alloca %struct.point467, align 4
  %4 = bitcast %struct.point467* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point467* %2 to i8*
  %6 = bitcast %struct.point467* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point467* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo468([2 x i64] %0) #0 {
  %2 = alloca %struct.point468, align 4
  %3 = alloca %struct.point468, align 4
  %4 = bitcast %struct.point468* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point468* %2 to i8*
  %6 = bitcast %struct.point468* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point468* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo469([2 x i64] %0) #0 {
  %2 = alloca %struct.point469, align 4
  %3 = alloca %struct.point469, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point469* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point469* %2 to i8*
  %9 = bitcast %struct.point469* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point469* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo470([2 x i64] %0) #0 {
  %2 = alloca %struct.point470, align 4
  %3 = alloca %struct.point470, align 4
  %4 = bitcast %struct.point470* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point470* %2 to i8*
  %6 = bitcast %struct.point470* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point470* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo471([2 x i64] %0) #0 {
  %2 = alloca %struct.point471, align 4
  %3 = alloca %struct.point471, align 4
  %4 = bitcast %struct.point471* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point471* %2 to i8*
  %6 = bitcast %struct.point471* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point471* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo472([2 x i64] %0) #0 {
  %2 = alloca %struct.point472, align 4
  %3 = alloca %struct.point472, align 4
  %4 = bitcast %struct.point472* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point472* %2 to i8*
  %6 = bitcast %struct.point472* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point472* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo473([2 x i64] %0) #0 {
  %2 = alloca %struct.point473, align 4
  %3 = alloca %struct.point473, align 4
  %4 = bitcast %struct.point473* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point473* %2 to i8*
  %6 = bitcast %struct.point473* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point473* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo474([2 x i64] %0) #0 {
  %2 = alloca %struct.point474, align 4
  %3 = alloca %struct.point474, align 4
  %4 = bitcast %struct.point474* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point474* %2 to i8*
  %6 = bitcast %struct.point474* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point474* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo475(%struct.point475* noalias sret(%struct.point475) align 4 %0, %struct.point475* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point475* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point475* %0 to i8*
  %6 = bitcast %struct.point475* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo476(%struct.point476* noalias sret(%struct.point476) align 4 %0, %struct.point476* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point476* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point476* %0 to i8*
  %6 = bitcast %struct.point476* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo477(%struct.point477* noalias sret(%struct.point477) align 4 %0, %struct.point477* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point477* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point477* %0 to i8*
  %6 = bitcast %struct.point477* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo478(%struct.point478* noalias sret(%struct.point478) align 4 %0, %struct.point478* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point478* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point478* %0 to i8*
  %6 = bitcast %struct.point478* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo479([2 x i64] %0) #0 {
  %2 = alloca %struct.point479, align 4
  %3 = alloca %struct.point479, align 4
  %4 = bitcast %struct.point479* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point479* %2 to i8*
  %6 = bitcast %struct.point479* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point479* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo480(%struct.point480* noalias sret(%struct.point480) align 4 %0, %struct.point480* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point480* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point480* %0 to i8*
  %6 = bitcast %struct.point480* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo481(%struct.point481* noalias sret(%struct.point481) align 4 %0, %struct.point481* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point481* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point481* %0 to i8*
  %6 = bitcast %struct.point481* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo482(%struct.point482* noalias sret(%struct.point482) align 4 %0, %struct.point482* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point482* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point482* %0 to i8*
  %6 = bitcast %struct.point482* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo483(%struct.point483* noalias sret(%struct.point483) align 4 %0, %struct.point483* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point483* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point483* %0 to i8*
  %6 = bitcast %struct.point483* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo484([2 x i64] %0) #0 {
  %2 = alloca %struct.point484, align 4
  %3 = alloca %struct.point484, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point484* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point484* %2 to i8*
  %9 = bitcast %struct.point484* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point484* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo485([2 x i64] %0) #0 {
  %2 = alloca %struct.point485, align 4
  %3 = alloca %struct.point485, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point485* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point485* %2 to i8*
  %9 = bitcast %struct.point485* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point485* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo486([2 x i64] %0) #0 {
  %2 = alloca %struct.point486, align 4
  %3 = alloca %struct.point486, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point486* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point486* %2 to i8*
  %9 = bitcast %struct.point486* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point486* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo487([2 x i64] %0) #0 {
  %2 = alloca %struct.point487, align 4
  %3 = alloca %struct.point487, align 4
  %4 = bitcast %struct.point487* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point487* %2 to i8*
  %6 = bitcast %struct.point487* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point487* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo488([2 x i64] %0) #0 {
  %2 = alloca %struct.point488, align 4
  %3 = alloca %struct.point488, align 4
  %4 = bitcast %struct.point488* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point488* %2 to i8*
  %6 = bitcast %struct.point488* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point488* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo489([2 x i64] %0) #0 {
  %2 = alloca %struct.point489, align 4
  %3 = alloca %struct.point489, align 4
  %4 = bitcast %struct.point489* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point489* %2 to i8*
  %6 = bitcast %struct.point489* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point489* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo490([2 x i64] %0) #0 {
  %2 = alloca %struct.point490, align 4
  %3 = alloca %struct.point490, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point490* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point490* %2 to i8*
  %9 = bitcast %struct.point490* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point490* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo491([2 x i64] %0) #0 {
  %2 = alloca %struct.point491, align 4
  %3 = alloca %struct.point491, align 4
  %4 = bitcast %struct.point491* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point491* %2 to i8*
  %6 = bitcast %struct.point491* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point491* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo492([2 x i64] %0) #0 {
  %2 = alloca %struct.point492, align 4
  %3 = alloca %struct.point492, align 4
  %4 = bitcast %struct.point492* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point492* %2 to i8*
  %6 = bitcast %struct.point492* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point492* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo493([2 x i64] %0) #0 {
  %2 = alloca %struct.point493, align 4
  %3 = alloca %struct.point493, align 4
  %4 = bitcast %struct.point493* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point493* %2 to i8*
  %6 = bitcast %struct.point493* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point493* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo494([2 x i64] %0) #0 {
  %2 = alloca %struct.point494, align 4
  %3 = alloca %struct.point494, align 4
  %4 = bitcast %struct.point494* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point494* %2 to i8*
  %6 = bitcast %struct.point494* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point494* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo495([2 x i64] %0) #0 {
  %2 = alloca %struct.point495, align 4
  %3 = alloca %struct.point495, align 4
  %4 = bitcast %struct.point495* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point495* %2 to i8*
  %6 = bitcast %struct.point495* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point495* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo496(%struct.point496* noalias sret(%struct.point496) align 4 %0, %struct.point496* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point496* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point496* %0 to i8*
  %6 = bitcast %struct.point496* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo497(%struct.point497* noalias sret(%struct.point497) align 4 %0, %struct.point497* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point497* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point497* %0 to i8*
  %6 = bitcast %struct.point497* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo498(%struct.point498* noalias sret(%struct.point498) align 4 %0, %struct.point498* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point498* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point498* %0 to i8*
  %6 = bitcast %struct.point498* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo499(%struct.point499* noalias sret(%struct.point499) align 4 %0, %struct.point499* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point499* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point499* %0 to i8*
  %6 = bitcast %struct.point499* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo500([2 x i64] %0) #0 {
  %2 = alloca %struct.point500, align 4
  %3 = alloca %struct.point500, align 4
  %4 = bitcast %struct.point500* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point500* %2 to i8*
  %6 = bitcast %struct.point500* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point500* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo501(%struct.point501* noalias sret(%struct.point501) align 4 %0, %struct.point501* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point501* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point501* %0 to i8*
  %6 = bitcast %struct.point501* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo502(%struct.point502* noalias sret(%struct.point502) align 4 %0, %struct.point502* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point502* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point502* %0 to i8*
  %6 = bitcast %struct.point502* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo503(%struct.point503* noalias sret(%struct.point503) align 4 %0, %struct.point503* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point503* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point503* %0 to i8*
  %6 = bitcast %struct.point503* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo504(%struct.point504* noalias sret(%struct.point504) align 4 %0, %struct.point504* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point504* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point504* %0 to i8*
  %6 = bitcast %struct.point504* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo505([2 x i64] %0) #0 {
  %2 = alloca %struct.point505, align 4
  %3 = alloca %struct.point505, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point505* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point505* %2 to i8*
  %9 = bitcast %struct.point505* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point505* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo506([2 x i64] %0) #0 {
  %2 = alloca %struct.point506, align 4
  %3 = alloca %struct.point506, align 4
  %4 = bitcast %struct.point506* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point506* %2 to i8*
  %6 = bitcast %struct.point506* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point506* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo507([2 x i64] %0) #0 {
  %2 = alloca %struct.point507, align 4
  %3 = alloca %struct.point507, align 4
  %4 = bitcast %struct.point507* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point507* %2 to i8*
  %6 = bitcast %struct.point507* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point507* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo508([2 x i64] %0) #0 {
  %2 = alloca %struct.point508, align 4
  %3 = alloca %struct.point508, align 4
  %4 = bitcast %struct.point508* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point508* %2 to i8*
  %6 = bitcast %struct.point508* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point508* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo509(%struct.point509* noalias sret(%struct.point509) align 4 %0, %struct.point509* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point509* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point509* %0 to i8*
  %6 = bitcast %struct.point509* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo510(%struct.point510* noalias sret(%struct.point510) align 4 %0, %struct.point510* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point510* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point510* %0 to i8*
  %6 = bitcast %struct.point510* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo511([2 x i64] %0) #0 {
  %2 = alloca %struct.point511, align 4
  %3 = alloca %struct.point511, align 4
  %4 = bitcast %struct.point511* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point511* %2 to i8*
  %6 = bitcast %struct.point511* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point511* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo512([2 x i64] %0) #0 {
  %2 = alloca %struct.point512, align 4
  %3 = alloca %struct.point512, align 4
  %4 = bitcast %struct.point512* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point512* %2 to i8*
  %6 = bitcast %struct.point512* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point512* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo513([2 x i64] %0) #0 {
  %2 = alloca %struct.point513, align 4
  %3 = alloca %struct.point513, align 4
  %4 = bitcast %struct.point513* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point513* %2 to i8*
  %6 = bitcast %struct.point513* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point513* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo514(%struct.point514* noalias sret(%struct.point514) align 4 %0, %struct.point514* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point514* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point514* %0 to i8*
  %6 = bitcast %struct.point514* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo515(%struct.point515* noalias sret(%struct.point515) align 4 %0, %struct.point515* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point515* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point515* %0 to i8*
  %6 = bitcast %struct.point515* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo516([2 x i64] %0) #0 {
  %2 = alloca %struct.point516, align 4
  %3 = alloca %struct.point516, align 4
  %4 = bitcast %struct.point516* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point516* %2 to i8*
  %6 = bitcast %struct.point516* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point516* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo517(%struct.point517* noalias sret(%struct.point517) align 4 %0, %struct.point517* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point517* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point517* %0 to i8*
  %6 = bitcast %struct.point517* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo518(%struct.point518* noalias sret(%struct.point518) align 4 %0, %struct.point518* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point518* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point518* %0 to i8*
  %6 = bitcast %struct.point518* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo519(%struct.point519* noalias sret(%struct.point519) align 4 %0, %struct.point519* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point519* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point519* %0 to i8*
  %6 = bitcast %struct.point519* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo520(%struct.point520* noalias sret(%struct.point520) align 4 %0, %struct.point520* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point520* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point520* %0 to i8*
  %6 = bitcast %struct.point520* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo521([2 x i64] %0) #0 {
  %2 = alloca %struct.point521, align 4
  %3 = alloca %struct.point521, align 4
  %4 = bitcast %struct.point521* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point521* %2 to i8*
  %6 = bitcast %struct.point521* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point521* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo522(%struct.point522* noalias sret(%struct.point522) align 4 %0, %struct.point522* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point522* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point522* %0 to i8*
  %6 = bitcast %struct.point522* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo523(%struct.point523* noalias sret(%struct.point523) align 4 %0, %struct.point523* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point523* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point523* %0 to i8*
  %6 = bitcast %struct.point523* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo524(%struct.point524* noalias sret(%struct.point524) align 4 %0, %struct.point524* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point524* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point524* %0 to i8*
  %6 = bitcast %struct.point524* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo525(%struct.point525* noalias sret(%struct.point525) align 4 %0, %struct.point525* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point525* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point525* %0 to i8*
  %6 = bitcast %struct.point525* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo526([2 x i64] %0) #0 {
  %2 = alloca %struct.point526, align 4
  %3 = alloca %struct.point526, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point526* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point526* %2 to i8*
  %9 = bitcast %struct.point526* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point526* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo527([2 x i64] %0) #0 {
  %2 = alloca %struct.point527, align 4
  %3 = alloca %struct.point527, align 4
  %4 = bitcast %struct.point527* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point527* %2 to i8*
  %6 = bitcast %struct.point527* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point527* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo528([2 x i64] %0) #0 {
  %2 = alloca %struct.point528, align 4
  %3 = alloca %struct.point528, align 4
  %4 = bitcast %struct.point528* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point528* %2 to i8*
  %6 = bitcast %struct.point528* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point528* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo529([2 x i64] %0) #0 {
  %2 = alloca %struct.point529, align 4
  %3 = alloca %struct.point529, align 4
  %4 = bitcast %struct.point529* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point529* %2 to i8*
  %6 = bitcast %struct.point529* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point529* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo530(%struct.point530* noalias sret(%struct.point530) align 4 %0, %struct.point530* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point530* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point530* %0 to i8*
  %6 = bitcast %struct.point530* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo531(%struct.point531* noalias sret(%struct.point531) align 4 %0, %struct.point531* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point531* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point531* %0 to i8*
  %6 = bitcast %struct.point531* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo532([2 x i64] %0) #0 {
  %2 = alloca %struct.point532, align 4
  %3 = alloca %struct.point532, align 4
  %4 = bitcast %struct.point532* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point532* %2 to i8*
  %6 = bitcast %struct.point532* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point532* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo533([2 x i64] %0) #0 {
  %2 = alloca %struct.point533, align 4
  %3 = alloca %struct.point533, align 4
  %4 = bitcast %struct.point533* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point533* %2 to i8*
  %6 = bitcast %struct.point533* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point533* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo534([2 x i64] %0) #0 {
  %2 = alloca %struct.point534, align 4
  %3 = alloca %struct.point534, align 4
  %4 = bitcast %struct.point534* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point534* %2 to i8*
  %6 = bitcast %struct.point534* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point534* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo535(%struct.point535* noalias sret(%struct.point535) align 4 %0, %struct.point535* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point535* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point535* %0 to i8*
  %6 = bitcast %struct.point535* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo536(%struct.point536* noalias sret(%struct.point536) align 4 %0, %struct.point536* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point536* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point536* %0 to i8*
  %6 = bitcast %struct.point536* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo537([2 x i64] %0) #0 {
  %2 = alloca %struct.point537, align 4
  %3 = alloca %struct.point537, align 4
  %4 = bitcast %struct.point537* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point537* %2 to i8*
  %6 = bitcast %struct.point537* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point537* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo538(%struct.point538* noalias sret(%struct.point538) align 4 %0, %struct.point538* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point538* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point538* %0 to i8*
  %6 = bitcast %struct.point538* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo539(%struct.point539* noalias sret(%struct.point539) align 4 %0, %struct.point539* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point539* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point539* %0 to i8*
  %6 = bitcast %struct.point539* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo540(%struct.point540* noalias sret(%struct.point540) align 4 %0, %struct.point540* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point540* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point540* %0 to i8*
  %6 = bitcast %struct.point540* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo541(%struct.point541* noalias sret(%struct.point541) align 4 %0, %struct.point541* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point541* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point541* %0 to i8*
  %6 = bitcast %struct.point541* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo542([2 x i64] %0) #0 {
  %2 = alloca %struct.point542, align 4
  %3 = alloca %struct.point542, align 4
  %4 = bitcast %struct.point542* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point542* %2 to i8*
  %6 = bitcast %struct.point542* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point542* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo543(%struct.point543* noalias sret(%struct.point543) align 4 %0, %struct.point543* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point543* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point543* %0 to i8*
  %6 = bitcast %struct.point543* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo544(%struct.point544* noalias sret(%struct.point544) align 4 %0, %struct.point544* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point544* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point544* %0 to i8*
  %6 = bitcast %struct.point544* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo545(%struct.point545* noalias sret(%struct.point545) align 4 %0, %struct.point545* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point545* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point545* %0 to i8*
  %6 = bitcast %struct.point545* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo546(%struct.point546* noalias sret(%struct.point546) align 4 %0, %struct.point546* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point546* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point546* %0 to i8*
  %6 = bitcast %struct.point546* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo547(i64 %0) #0 {
  %2 = alloca %struct.point547, align 4
  %3 = alloca %struct.point547, align 4
  %4 = bitcast %struct.point547* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point547* %2 to i8*
  %6 = bitcast %struct.point547* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point547* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo548(i64 %0) #0 {
  %2 = alloca %struct.point548, align 4
  %3 = alloca %struct.point548, align 4
  %4 = bitcast %struct.point548* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point548* %2 to i8*
  %6 = bitcast %struct.point548* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point548* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo549(i64 %0) #0 {
  %2 = alloca %struct.point549, align 4
  %3 = alloca %struct.point549, align 4
  %4 = bitcast %struct.point549* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point549* %2 to i8*
  %6 = bitcast %struct.point549* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point549* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo550([2 x i64] %0) #0 {
  %2 = alloca %struct.point550, align 4
  %3 = alloca %struct.point550, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point550* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point550* %2 to i8*
  %9 = bitcast %struct.point550* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point550* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo551([2 x i64] %0) #0 {
  %2 = alloca %struct.point551, align 4
  %3 = alloca %struct.point551, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point551* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point551* %2 to i8*
  %9 = bitcast %struct.point551* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point551* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo552([2 x i64] %0) #0 {
  %2 = alloca %struct.point552, align 4
  %3 = alloca %struct.point552, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point552* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point552* %2 to i8*
  %9 = bitcast %struct.point552* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point552* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo553(i64 %0) #0 {
  %2 = alloca %struct.point553, align 4
  %3 = alloca %struct.point553, align 4
  %4 = bitcast %struct.point553* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point553* %2 to i8*
  %6 = bitcast %struct.point553* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point553* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo554([2 x i64] %0) #0 {
  %2 = alloca %struct.point554, align 4
  %3 = alloca %struct.point554, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point554* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point554* %2 to i8*
  %9 = bitcast %struct.point554* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point554* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo555([2 x i64] %0) #0 {
  %2 = alloca %struct.point555, align 4
  %3 = alloca %struct.point555, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point555* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point555* %2 to i8*
  %9 = bitcast %struct.point555* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point555* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo556([2 x i64] %0) #0 {
  %2 = alloca %struct.point556, align 4
  %3 = alloca %struct.point556, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point556* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point556* %2 to i8*
  %9 = bitcast %struct.point556* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point556* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo557([2 x i64] %0) #0 {
  %2 = alloca %struct.point557, align 4
  %3 = alloca %struct.point557, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point557* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point557* %2 to i8*
  %9 = bitcast %struct.point557* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point557* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo558([2 x i64] %0) #0 {
  %2 = alloca %struct.point558, align 4
  %3 = alloca %struct.point558, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point558* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point558* %2 to i8*
  %9 = bitcast %struct.point558* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point558* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo559([2 x i64] %0) #0 {
  %2 = alloca %struct.point559, align 4
  %3 = alloca %struct.point559, align 4
  %4 = bitcast %struct.point559* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point559* %2 to i8*
  %6 = bitcast %struct.point559* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point559* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo560([2 x i64] %0) #0 {
  %2 = alloca %struct.point560, align 4
  %3 = alloca %struct.point560, align 4
  %4 = bitcast %struct.point560* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point560* %2 to i8*
  %6 = bitcast %struct.point560* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point560* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo561([2 x i64] %0) #0 {
  %2 = alloca %struct.point561, align 4
  %3 = alloca %struct.point561, align 4
  %4 = bitcast %struct.point561* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point561* %2 to i8*
  %6 = bitcast %struct.point561* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point561* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo562([2 x i64] %0) #0 {
  %2 = alloca %struct.point562, align 4
  %3 = alloca %struct.point562, align 4
  %4 = bitcast %struct.point562* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point562* %2 to i8*
  %6 = bitcast %struct.point562* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point562* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo563([2 x i64] %0) #0 {
  %2 = alloca %struct.point563, align 4
  %3 = alloca %struct.point563, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point563* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point563* %2 to i8*
  %9 = bitcast %struct.point563* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point563* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo564([2 x i64] %0) #0 {
  %2 = alloca %struct.point564, align 4
  %3 = alloca %struct.point564, align 4
  %4 = bitcast %struct.point564* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point564* %2 to i8*
  %6 = bitcast %struct.point564* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point564* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo565([2 x i64] %0) #0 {
  %2 = alloca %struct.point565, align 4
  %3 = alloca %struct.point565, align 4
  %4 = bitcast %struct.point565* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point565* %2 to i8*
  %6 = bitcast %struct.point565* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point565* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo566([2 x i64] %0) #0 {
  %2 = alloca %struct.point566, align 4
  %3 = alloca %struct.point566, align 4
  %4 = bitcast %struct.point566* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point566* %2 to i8*
  %6 = bitcast %struct.point566* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point566* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo567([2 x i64] %0) #0 {
  %2 = alloca %struct.point567, align 4
  %3 = alloca %struct.point567, align 4
  %4 = bitcast %struct.point567* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point567* %2 to i8*
  %6 = bitcast %struct.point567* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point567* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo568(i64 %0) #0 {
  %2 = alloca %struct.point568, align 4
  %3 = alloca %struct.point568, align 4
  %4 = bitcast %struct.point568* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point568* %2 to i8*
  %6 = bitcast %struct.point568* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point568* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo569([2 x i64] %0) #0 {
  %2 = alloca %struct.point569, align 4
  %3 = alloca %struct.point569, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point569* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point569* %2 to i8*
  %9 = bitcast %struct.point569* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point569* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo570([2 x i64] %0) #0 {
  %2 = alloca %struct.point570, align 4
  %3 = alloca %struct.point570, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point570* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point570* %2 to i8*
  %9 = bitcast %struct.point570* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point570* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo571([2 x i64] %0) #0 {
  %2 = alloca %struct.point571, align 4
  %3 = alloca %struct.point571, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point571* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point571* %2 to i8*
  %9 = bitcast %struct.point571* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point571* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo572([2 x i64] %0) #0 {
  %2 = alloca %struct.point572, align 4
  %3 = alloca %struct.point572, align 4
  %4 = bitcast %struct.point572* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point572* %2 to i8*
  %6 = bitcast %struct.point572* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point572* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo573([2 x i64] %0) #0 {
  %2 = alloca %struct.point573, align 4
  %3 = alloca %struct.point573, align 4
  %4 = bitcast %struct.point573* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point573* %2 to i8*
  %6 = bitcast %struct.point573* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point573* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo574([2 x i64] %0) #0 {
  %2 = alloca %struct.point574, align 4
  %3 = alloca %struct.point574, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point574* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point574* %2 to i8*
  %9 = bitcast %struct.point574* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point574* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo575([2 x i64] %0) #0 {
  %2 = alloca %struct.point575, align 4
  %3 = alloca %struct.point575, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point575* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point575* %2 to i8*
  %9 = bitcast %struct.point575* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point575* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo576([2 x i64] %0) #0 {
  %2 = alloca %struct.point576, align 4
  %3 = alloca %struct.point576, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point576* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point576* %2 to i8*
  %9 = bitcast %struct.point576* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point576* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo577([2 x i64] %0) #0 {
  %2 = alloca %struct.point577, align 4
  %3 = alloca %struct.point577, align 4
  %4 = bitcast %struct.point577* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point577* %2 to i8*
  %6 = bitcast %struct.point577* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point577* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo578([2 x i64] %0) #0 {
  %2 = alloca %struct.point578, align 4
  %3 = alloca %struct.point578, align 4
  %4 = bitcast %struct.point578* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point578* %2 to i8*
  %6 = bitcast %struct.point578* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point578* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo579([2 x i64] %0) #0 {
  %2 = alloca %struct.point579, align 4
  %3 = alloca %struct.point579, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point579* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point579* %2 to i8*
  %9 = bitcast %struct.point579* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point579* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo580([2 x i64] %0) #0 {
  %2 = alloca %struct.point580, align 4
  %3 = alloca %struct.point580, align 4
  %4 = bitcast %struct.point580* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point580* %2 to i8*
  %6 = bitcast %struct.point580* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point580* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo581([2 x i64] %0) #0 {
  %2 = alloca %struct.point581, align 4
  %3 = alloca %struct.point581, align 4
  %4 = bitcast %struct.point581* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point581* %2 to i8*
  %6 = bitcast %struct.point581* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point581* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo582([2 x i64] %0) #0 {
  %2 = alloca %struct.point582, align 4
  %3 = alloca %struct.point582, align 4
  %4 = bitcast %struct.point582* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point582* %2 to i8*
  %6 = bitcast %struct.point582* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point582* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo583([2 x i64] %0) #0 {
  %2 = alloca %struct.point583, align 4
  %3 = alloca %struct.point583, align 4
  %4 = bitcast %struct.point583* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point583* %2 to i8*
  %6 = bitcast %struct.point583* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point583* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo584([2 x i64] %0) #0 {
  %2 = alloca %struct.point584, align 4
  %3 = alloca %struct.point584, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point584* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point584* %2 to i8*
  %9 = bitcast %struct.point584* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point584* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo585([2 x i64] %0) #0 {
  %2 = alloca %struct.point585, align 4
  %3 = alloca %struct.point585, align 4
  %4 = bitcast %struct.point585* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point585* %2 to i8*
  %6 = bitcast %struct.point585* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point585* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo586([2 x i64] %0) #0 {
  %2 = alloca %struct.point586, align 4
  %3 = alloca %struct.point586, align 4
  %4 = bitcast %struct.point586* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point586* %2 to i8*
  %6 = bitcast %struct.point586* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point586* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo587([2 x i64] %0) #0 {
  %2 = alloca %struct.point587, align 4
  %3 = alloca %struct.point587, align 4
  %4 = bitcast %struct.point587* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point587* %2 to i8*
  %6 = bitcast %struct.point587* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point587* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo588([2 x i64] %0) #0 {
  %2 = alloca %struct.point588, align 4
  %3 = alloca %struct.point588, align 4
  %4 = bitcast %struct.point588* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point588* %2 to i8*
  %6 = bitcast %struct.point588* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point588* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo589([2 x i64] %0) #0 {
  %2 = alloca %struct.point589, align 4
  %3 = alloca %struct.point589, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point589* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point589* %2 to i8*
  %9 = bitcast %struct.point589* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point589* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo590([2 x i64] %0) #0 {
  %2 = alloca %struct.point590, align 4
  %3 = alloca %struct.point590, align 4
  %4 = bitcast %struct.point590* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point590* %2 to i8*
  %6 = bitcast %struct.point590* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point590* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo591([2 x i64] %0) #0 {
  %2 = alloca %struct.point591, align 4
  %3 = alloca %struct.point591, align 4
  %4 = bitcast %struct.point591* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point591* %2 to i8*
  %6 = bitcast %struct.point591* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point591* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo592([2 x i64] %0) #0 {
  %2 = alloca %struct.point592, align 4
  %3 = alloca %struct.point592, align 4
  %4 = bitcast %struct.point592* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point592* %2 to i8*
  %6 = bitcast %struct.point592* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point592* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo593(%struct.point593* noalias sret(%struct.point593) align 4 %0, %struct.point593* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point593* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point593* %0 to i8*
  %6 = bitcast %struct.point593* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo594(%struct.point594* noalias sret(%struct.point594) align 4 %0, %struct.point594* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point594* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point594* %0 to i8*
  %6 = bitcast %struct.point594* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo595([2 x i64] %0) #0 {
  %2 = alloca %struct.point595, align 4
  %3 = alloca %struct.point595, align 4
  %4 = bitcast %struct.point595* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point595* %2 to i8*
  %6 = bitcast %struct.point595* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point595* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo596([2 x i64] %0) #0 {
  %2 = alloca %struct.point596, align 4
  %3 = alloca %struct.point596, align 4
  %4 = bitcast %struct.point596* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point596* %2 to i8*
  %6 = bitcast %struct.point596* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point596* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo597([2 x i64] %0) #0 {
  %2 = alloca %struct.point597, align 4
  %3 = alloca %struct.point597, align 4
  %4 = bitcast %struct.point597* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point597* %2 to i8*
  %6 = bitcast %struct.point597* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point597* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo598(%struct.point598* noalias sret(%struct.point598) align 4 %0, %struct.point598* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point598* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point598* %0 to i8*
  %6 = bitcast %struct.point598* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo599(%struct.point599* noalias sret(%struct.point599) align 4 %0, %struct.point599* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point599* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point599* %0 to i8*
  %6 = bitcast %struct.point599* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo600([2 x i64] %0) #0 {
  %2 = alloca %struct.point600, align 4
  %3 = alloca %struct.point600, align 4
  %4 = bitcast %struct.point600* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point600* %2 to i8*
  %6 = bitcast %struct.point600* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point600* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo601(%struct.point601* noalias sret(%struct.point601) align 4 %0, %struct.point601* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point601* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point601* %0 to i8*
  %6 = bitcast %struct.point601* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo602(%struct.point602* noalias sret(%struct.point602) align 4 %0, %struct.point602* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point602* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point602* %0 to i8*
  %6 = bitcast %struct.point602* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo603(%struct.point603* noalias sret(%struct.point603) align 4 %0, %struct.point603* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point603* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point603* %0 to i8*
  %6 = bitcast %struct.point603* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo604(%struct.point604* noalias sret(%struct.point604) align 4 %0, %struct.point604* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point604* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point604* %0 to i8*
  %6 = bitcast %struct.point604* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo605([2 x i64] %0) #0 {
  %2 = alloca %struct.point605, align 4
  %3 = alloca %struct.point605, align 4
  %4 = bitcast %struct.point605* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point605* %2 to i8*
  %6 = bitcast %struct.point605* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point605* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo606(%struct.point606* noalias sret(%struct.point606) align 4 %0, %struct.point606* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point606* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point606* %0 to i8*
  %6 = bitcast %struct.point606* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo607(%struct.point607* noalias sret(%struct.point607) align 4 %0, %struct.point607* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point607* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point607* %0 to i8*
  %6 = bitcast %struct.point607* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo608(%struct.point608* noalias sret(%struct.point608) align 4 %0, %struct.point608* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point608* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point608* %0 to i8*
  %6 = bitcast %struct.point608* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo609(%struct.point609* noalias sret(%struct.point609) align 4 %0, %struct.point609* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point609* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point609* %0 to i8*
  %6 = bitcast %struct.point609* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo610([2 x i64] %0) #0 {
  %2 = alloca %struct.point610, align 4
  %3 = alloca %struct.point610, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point610* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point610* %2 to i8*
  %9 = bitcast %struct.point610* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point610* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo611([2 x i64] %0) #0 {
  %2 = alloca %struct.point611, align 4
  %3 = alloca %struct.point611, align 4
  %4 = bitcast %struct.point611* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point611* %2 to i8*
  %6 = bitcast %struct.point611* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point611* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo612([2 x i64] %0) #0 {
  %2 = alloca %struct.point612, align 4
  %3 = alloca %struct.point612, align 4
  %4 = bitcast %struct.point612* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point612* %2 to i8*
  %6 = bitcast %struct.point612* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point612* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo613([2 x i64] %0) #0 {
  %2 = alloca %struct.point613, align 4
  %3 = alloca %struct.point613, align 4
  %4 = bitcast %struct.point613* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point613* %2 to i8*
  %6 = bitcast %struct.point613* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point613* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo614(%struct.point614* noalias sret(%struct.point614) align 4 %0, %struct.point614* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point614* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point614* %0 to i8*
  %6 = bitcast %struct.point614* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo615(%struct.point615* noalias sret(%struct.point615) align 4 %0, %struct.point615* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point615* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point615* %0 to i8*
  %6 = bitcast %struct.point615* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo616([2 x i64] %0) #0 {
  %2 = alloca %struct.point616, align 4
  %3 = alloca %struct.point616, align 4
  %4 = bitcast %struct.point616* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point616* %2 to i8*
  %6 = bitcast %struct.point616* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point616* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo617([2 x i64] %0) #0 {
  %2 = alloca %struct.point617, align 4
  %3 = alloca %struct.point617, align 4
  %4 = bitcast %struct.point617* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point617* %2 to i8*
  %6 = bitcast %struct.point617* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point617* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo618([2 x i64] %0) #0 {
  %2 = alloca %struct.point618, align 4
  %3 = alloca %struct.point618, align 4
  %4 = bitcast %struct.point618* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point618* %2 to i8*
  %6 = bitcast %struct.point618* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point618* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo619(%struct.point619* noalias sret(%struct.point619) align 4 %0, %struct.point619* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point619* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point619* %0 to i8*
  %6 = bitcast %struct.point619* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo620(%struct.point620* noalias sret(%struct.point620) align 4 %0, %struct.point620* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point620* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point620* %0 to i8*
  %6 = bitcast %struct.point620* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo621([2 x i64] %0) #0 {
  %2 = alloca %struct.point621, align 4
  %3 = alloca %struct.point621, align 4
  %4 = bitcast %struct.point621* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point621* %2 to i8*
  %6 = bitcast %struct.point621* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point621* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo622(%struct.point622* noalias sret(%struct.point622) align 4 %0, %struct.point622* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point622* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point622* %0 to i8*
  %6 = bitcast %struct.point622* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo623(%struct.point623* noalias sret(%struct.point623) align 4 %0, %struct.point623* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point623* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point623* %0 to i8*
  %6 = bitcast %struct.point623* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo624(%struct.point624* noalias sret(%struct.point624) align 4 %0, %struct.point624* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point624* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point624* %0 to i8*
  %6 = bitcast %struct.point624* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo625(%struct.point625* noalias sret(%struct.point625) align 4 %0, %struct.point625* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point625* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point625* %0 to i8*
  %6 = bitcast %struct.point625* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo626([2 x i64] %0) #0 {
  %2 = alloca %struct.point626, align 4
  %3 = alloca %struct.point626, align 4
  %4 = bitcast %struct.point626* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point626* %2 to i8*
  %6 = bitcast %struct.point626* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point626* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo627(%struct.point627* noalias sret(%struct.point627) align 4 %0, %struct.point627* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point627* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point627* %0 to i8*
  %6 = bitcast %struct.point627* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo628(%struct.point628* noalias sret(%struct.point628) align 4 %0, %struct.point628* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point628* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point628* %0 to i8*
  %6 = bitcast %struct.point628* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo629(%struct.point629* noalias sret(%struct.point629) align 4 %0, %struct.point629* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point629* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point629* %0 to i8*
  %6 = bitcast %struct.point629* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo630(%struct.point630* noalias sret(%struct.point630) align 4 %0, %struct.point630* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point630* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point630* %0 to i8*
  %6 = bitcast %struct.point630* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo631(i64 %0) #0 {
  %2 = alloca %struct.point631, align 4
  %3 = alloca %struct.point631, align 4
  %4 = bitcast %struct.point631* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point631* %2 to i8*
  %6 = bitcast %struct.point631* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point631* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo632(i64 %0) #0 {
  %2 = alloca %struct.point632, align 4
  %3 = alloca %struct.point632, align 4
  %4 = bitcast %struct.point632* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point632* %2 to i8*
  %6 = bitcast %struct.point632* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point632* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo633([2 x i64] %0) #0 {
  %2 = alloca %struct.point633, align 4
  %3 = alloca %struct.point633, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point633* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point633* %2 to i8*
  %9 = bitcast %struct.point633* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point633* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo634([2 x i64] %0) #0 {
  %2 = alloca %struct.point634, align 4
  %3 = alloca %struct.point634, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point634* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point634* %2 to i8*
  %9 = bitcast %struct.point634* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point634* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo635([2 x i64] %0) #0 {
  %2 = alloca %struct.point635, align 4
  %3 = alloca %struct.point635, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point635* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point635* %2 to i8*
  %9 = bitcast %struct.point635* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point635* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo636([2 x i64] %0) #0 {
  %2 = alloca %struct.point636, align 4
  %3 = alloca %struct.point636, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point636* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point636* %2 to i8*
  %9 = bitcast %struct.point636* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point636* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo637([2 x i64] %0) #0 {
  %2 = alloca %struct.point637, align 4
  %3 = alloca %struct.point637, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point637* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point637* %2 to i8*
  %9 = bitcast %struct.point637* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point637* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo638([2 x i64] %0) #0 {
  %2 = alloca %struct.point638, align 4
  %3 = alloca %struct.point638, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point638* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point638* %2 to i8*
  %9 = bitcast %struct.point638* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point638* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo639([2 x i64] %0) #0 {
  %2 = alloca %struct.point639, align 4
  %3 = alloca %struct.point639, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point639* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point639* %2 to i8*
  %9 = bitcast %struct.point639* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point639* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo640([2 x i64] %0) #0 {
  %2 = alloca %struct.point640, align 4
  %3 = alloca %struct.point640, align 4
  %4 = bitcast %struct.point640* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point640* %2 to i8*
  %6 = bitcast %struct.point640* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point640* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo641([2 x i64] %0) #0 {
  %2 = alloca %struct.point641, align 4
  %3 = alloca %struct.point641, align 4
  %4 = bitcast %struct.point641* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point641* %2 to i8*
  %6 = bitcast %struct.point641* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point641* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo642([2 x i64] %0) #0 {
  %2 = alloca %struct.point642, align 4
  %3 = alloca %struct.point642, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point642* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point642* %2 to i8*
  %9 = bitcast %struct.point642* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point642* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo643([2 x i64] %0) #0 {
  %2 = alloca %struct.point643, align 4
  %3 = alloca %struct.point643, align 4
  %4 = bitcast %struct.point643* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point643* %2 to i8*
  %6 = bitcast %struct.point643* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point643* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo644([2 x i64] %0) #0 {
  %2 = alloca %struct.point644, align 4
  %3 = alloca %struct.point644, align 4
  %4 = bitcast %struct.point644* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point644* %2 to i8*
  %6 = bitcast %struct.point644* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point644* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo645([2 x i64] %0) #0 {
  %2 = alloca %struct.point645, align 4
  %3 = alloca %struct.point645, align 4
  %4 = bitcast %struct.point645* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point645* %2 to i8*
  %6 = bitcast %struct.point645* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point645* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo646([2 x i64] %0) #0 {
  %2 = alloca %struct.point646, align 4
  %3 = alloca %struct.point646, align 4
  %4 = bitcast %struct.point646* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point646* %2 to i8*
  %6 = bitcast %struct.point646* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point646* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo647([2 x i64] %0) #0 {
  %2 = alloca %struct.point647, align 4
  %3 = alloca %struct.point647, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point647* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point647* %2 to i8*
  %9 = bitcast %struct.point647* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point647* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo648([2 x i64] %0) #0 {
  %2 = alloca %struct.point648, align 4
  %3 = alloca %struct.point648, align 4
  %4 = bitcast %struct.point648* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point648* %2 to i8*
  %6 = bitcast %struct.point648* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point648* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo649([2 x i64] %0) #0 {
  %2 = alloca %struct.point649, align 4
  %3 = alloca %struct.point649, align 4
  %4 = bitcast %struct.point649* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point649* %2 to i8*
  %6 = bitcast %struct.point649* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point649* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo650([2 x i64] %0) #0 {
  %2 = alloca %struct.point650, align 4
  %3 = alloca %struct.point650, align 4
  %4 = bitcast %struct.point650* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point650* %2 to i8*
  %6 = bitcast %struct.point650* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point650* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo651([2 x i64] %0) #0 {
  %2 = alloca %struct.point651, align 4
  %3 = alloca %struct.point651, align 4
  %4 = bitcast %struct.point651* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point651* %2 to i8*
  %6 = bitcast %struct.point651* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point651* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo652(i64 %0) #0 {
  %2 = alloca %struct.point652, align 4
  %3 = alloca %struct.point652, align 4
  %4 = bitcast %struct.point652* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point652* %2 to i8*
  %6 = bitcast %struct.point652* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point652* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo653([2 x i64] %0) #0 {
  %2 = alloca %struct.point653, align 4
  %3 = alloca %struct.point653, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point653* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point653* %2 to i8*
  %9 = bitcast %struct.point653* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point653* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo654([2 x i64] %0) #0 {
  %2 = alloca %struct.point654, align 4
  %3 = alloca %struct.point654, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point654* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point654* %2 to i8*
  %9 = bitcast %struct.point654* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point654* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo655([2 x i64] %0) #0 {
  %2 = alloca %struct.point655, align 4
  %3 = alloca %struct.point655, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point655* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point655* %2 to i8*
  %9 = bitcast %struct.point655* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point655* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo656([2 x i64] %0) #0 {
  %2 = alloca %struct.point656, align 4
  %3 = alloca %struct.point656, align 4
  %4 = bitcast %struct.point656* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point656* %2 to i8*
  %6 = bitcast %struct.point656* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point656* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo657([2 x i64] %0) #0 {
  %2 = alloca %struct.point657, align 4
  %3 = alloca %struct.point657, align 4
  %4 = bitcast %struct.point657* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point657* %2 to i8*
  %6 = bitcast %struct.point657* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point657* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo658([2 x i64] %0) #0 {
  %2 = alloca %struct.point658, align 4
  %3 = alloca %struct.point658, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point658* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point658* %2 to i8*
  %9 = bitcast %struct.point658* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point658* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo659([2 x i64] %0) #0 {
  %2 = alloca %struct.point659, align 4
  %3 = alloca %struct.point659, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point659* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point659* %2 to i8*
  %9 = bitcast %struct.point659* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point659* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo660([2 x i64] %0) #0 {
  %2 = alloca %struct.point660, align 4
  %3 = alloca %struct.point660, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point660* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point660* %2 to i8*
  %9 = bitcast %struct.point660* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point660* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo661([2 x i64] %0) #0 {
  %2 = alloca %struct.point661, align 4
  %3 = alloca %struct.point661, align 4
  %4 = bitcast %struct.point661* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point661* %2 to i8*
  %6 = bitcast %struct.point661* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point661* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo662([2 x i64] %0) #0 {
  %2 = alloca %struct.point662, align 4
  %3 = alloca %struct.point662, align 4
  %4 = bitcast %struct.point662* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point662* %2 to i8*
  %6 = bitcast %struct.point662* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point662* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo663([2 x i64] %0) #0 {
  %2 = alloca %struct.point663, align 4
  %3 = alloca %struct.point663, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point663* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point663* %2 to i8*
  %9 = bitcast %struct.point663* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point663* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo664([2 x i64] %0) #0 {
  %2 = alloca %struct.point664, align 4
  %3 = alloca %struct.point664, align 4
  %4 = bitcast %struct.point664* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point664* %2 to i8*
  %6 = bitcast %struct.point664* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point664* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo665([2 x i64] %0) #0 {
  %2 = alloca %struct.point665, align 4
  %3 = alloca %struct.point665, align 4
  %4 = bitcast %struct.point665* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point665* %2 to i8*
  %6 = bitcast %struct.point665* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point665* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo666([2 x i64] %0) #0 {
  %2 = alloca %struct.point666, align 4
  %3 = alloca %struct.point666, align 4
  %4 = bitcast %struct.point666* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point666* %2 to i8*
  %6 = bitcast %struct.point666* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point666* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo667([2 x i64] %0) #0 {
  %2 = alloca %struct.point667, align 4
  %3 = alloca %struct.point667, align 4
  %4 = bitcast %struct.point667* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point667* %2 to i8*
  %6 = bitcast %struct.point667* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point667* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo668([2 x i64] %0) #0 {
  %2 = alloca %struct.point668, align 4
  %3 = alloca %struct.point668, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point668* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point668* %2 to i8*
  %9 = bitcast %struct.point668* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point668* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo669([2 x i64] %0) #0 {
  %2 = alloca %struct.point669, align 4
  %3 = alloca %struct.point669, align 4
  %4 = bitcast %struct.point669* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point669* %2 to i8*
  %6 = bitcast %struct.point669* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point669* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo670([2 x i64] %0) #0 {
  %2 = alloca %struct.point670, align 4
  %3 = alloca %struct.point670, align 4
  %4 = bitcast %struct.point670* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point670* %2 to i8*
  %6 = bitcast %struct.point670* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point670* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo671([2 x i64] %0) #0 {
  %2 = alloca %struct.point671, align 4
  %3 = alloca %struct.point671, align 4
  %4 = bitcast %struct.point671* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point671* %2 to i8*
  %6 = bitcast %struct.point671* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point671* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo672([2 x i64] %0) #0 {
  %2 = alloca %struct.point672, align 4
  %3 = alloca %struct.point672, align 4
  %4 = bitcast %struct.point672* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point672* %2 to i8*
  %6 = bitcast %struct.point672* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point672* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo673([2 x i64] %0) #0 {
  %2 = alloca %struct.point673, align 4
  %3 = alloca %struct.point673, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point673* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point673* %2 to i8*
  %9 = bitcast %struct.point673* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point673* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo674([2 x i64] %0) #0 {
  %2 = alloca %struct.point674, align 4
  %3 = alloca %struct.point674, align 4
  %4 = bitcast %struct.point674* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point674* %2 to i8*
  %6 = bitcast %struct.point674* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point674* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo675([2 x i64] %0) #0 {
  %2 = alloca %struct.point675, align 4
  %3 = alloca %struct.point675, align 4
  %4 = bitcast %struct.point675* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point675* %2 to i8*
  %6 = bitcast %struct.point675* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point675* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo676([2 x i64] %0) #0 {
  %2 = alloca %struct.point676, align 4
  %3 = alloca %struct.point676, align 4
  %4 = bitcast %struct.point676* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point676* %2 to i8*
  %6 = bitcast %struct.point676* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point676* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo677(%struct.point677* noalias sret(%struct.point677) align 4 %0, %struct.point677* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point677* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point677* %0 to i8*
  %6 = bitcast %struct.point677* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo678(%struct.point678* noalias sret(%struct.point678) align 4 %0, %struct.point678* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point678* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point678* %0 to i8*
  %6 = bitcast %struct.point678* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo679([2 x i64] %0) #0 {
  %2 = alloca %struct.point679, align 4
  %3 = alloca %struct.point679, align 4
  %4 = bitcast %struct.point679* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point679* %2 to i8*
  %6 = bitcast %struct.point679* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point679* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo680([2 x i64] %0) #0 {
  %2 = alloca %struct.point680, align 4
  %3 = alloca %struct.point680, align 4
  %4 = bitcast %struct.point680* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point680* %2 to i8*
  %6 = bitcast %struct.point680* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point680* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo681([2 x i64] %0) #0 {
  %2 = alloca %struct.point681, align 4
  %3 = alloca %struct.point681, align 4
  %4 = bitcast %struct.point681* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point681* %2 to i8*
  %6 = bitcast %struct.point681* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point681* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo682(%struct.point682* noalias sret(%struct.point682) align 4 %0, %struct.point682* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point682* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point682* %0 to i8*
  %6 = bitcast %struct.point682* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo683(%struct.point683* noalias sret(%struct.point683) align 4 %0, %struct.point683* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point683* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point683* %0 to i8*
  %6 = bitcast %struct.point683* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo684([2 x i64] %0) #0 {
  %2 = alloca %struct.point684, align 4
  %3 = alloca %struct.point684, align 4
  %4 = bitcast %struct.point684* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point684* %2 to i8*
  %6 = bitcast %struct.point684* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point684* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo685(%struct.point685* noalias sret(%struct.point685) align 4 %0, %struct.point685* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point685* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point685* %0 to i8*
  %6 = bitcast %struct.point685* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo686(%struct.point686* noalias sret(%struct.point686) align 4 %0, %struct.point686* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point686* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point686* %0 to i8*
  %6 = bitcast %struct.point686* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo687(%struct.point687* noalias sret(%struct.point687) align 4 %0, %struct.point687* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point687* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point687* %0 to i8*
  %6 = bitcast %struct.point687* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo688(%struct.point688* noalias sret(%struct.point688) align 4 %0, %struct.point688* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point688* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point688* %0 to i8*
  %6 = bitcast %struct.point688* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo689([2 x i64] %0) #0 {
  %2 = alloca %struct.point689, align 4
  %3 = alloca %struct.point689, align 4
  %4 = bitcast %struct.point689* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point689* %2 to i8*
  %6 = bitcast %struct.point689* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point689* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo690(%struct.point690* noalias sret(%struct.point690) align 4 %0, %struct.point690* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point690* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point690* %0 to i8*
  %6 = bitcast %struct.point690* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo691(%struct.point691* noalias sret(%struct.point691) align 4 %0, %struct.point691* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point691* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point691* %0 to i8*
  %6 = bitcast %struct.point691* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo692(%struct.point692* noalias sret(%struct.point692) align 4 %0, %struct.point692* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point692* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point692* %0 to i8*
  %6 = bitcast %struct.point692* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo693(%struct.point693* noalias sret(%struct.point693) align 4 %0, %struct.point693* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point693* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point693* %0 to i8*
  %6 = bitcast %struct.point693* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo694([2 x i64] %0) #0 {
  %2 = alloca %struct.point694, align 4
  %3 = alloca %struct.point694, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point694* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point694* %2 to i8*
  %9 = bitcast %struct.point694* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point694* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo695([2 x i64] %0) #0 {
  %2 = alloca %struct.point695, align 4
  %3 = alloca %struct.point695, align 4
  %4 = bitcast %struct.point695* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point695* %2 to i8*
  %6 = bitcast %struct.point695* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point695* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo696([2 x i64] %0) #0 {
  %2 = alloca %struct.point696, align 4
  %3 = alloca %struct.point696, align 4
  %4 = bitcast %struct.point696* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point696* %2 to i8*
  %6 = bitcast %struct.point696* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point696* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo697([2 x i64] %0) #0 {
  %2 = alloca %struct.point697, align 4
  %3 = alloca %struct.point697, align 4
  %4 = bitcast %struct.point697* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point697* %2 to i8*
  %6 = bitcast %struct.point697* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point697* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo698(%struct.point698* noalias sret(%struct.point698) align 4 %0, %struct.point698* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point698* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point698* %0 to i8*
  %6 = bitcast %struct.point698* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo699(%struct.point699* noalias sret(%struct.point699) align 4 %0, %struct.point699* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point699* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point699* %0 to i8*
  %6 = bitcast %struct.point699* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo700([2 x i64] %0) #0 {
  %2 = alloca %struct.point700, align 4
  %3 = alloca %struct.point700, align 4
  %4 = bitcast %struct.point700* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point700* %2 to i8*
  %6 = bitcast %struct.point700* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point700* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo701([2 x i64] %0) #0 {
  %2 = alloca %struct.point701, align 4
  %3 = alloca %struct.point701, align 4
  %4 = bitcast %struct.point701* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point701* %2 to i8*
  %6 = bitcast %struct.point701* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point701* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo702([2 x i64] %0) #0 {
  %2 = alloca %struct.point702, align 4
  %3 = alloca %struct.point702, align 4
  %4 = bitcast %struct.point702* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point702* %2 to i8*
  %6 = bitcast %struct.point702* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point702* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo703(%struct.point703* noalias sret(%struct.point703) align 4 %0, %struct.point703* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point703* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point703* %0 to i8*
  %6 = bitcast %struct.point703* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo704(%struct.point704* noalias sret(%struct.point704) align 4 %0, %struct.point704* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point704* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point704* %0 to i8*
  %6 = bitcast %struct.point704* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo705([2 x i64] %0) #0 {
  %2 = alloca %struct.point705, align 4
  %3 = alloca %struct.point705, align 4
  %4 = bitcast %struct.point705* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point705* %2 to i8*
  %6 = bitcast %struct.point705* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point705* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo706(%struct.point706* noalias sret(%struct.point706) align 4 %0, %struct.point706* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point706* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point706* %0 to i8*
  %6 = bitcast %struct.point706* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo707(%struct.point707* noalias sret(%struct.point707) align 4 %0, %struct.point707* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point707* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point707* %0 to i8*
  %6 = bitcast %struct.point707* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo708(%struct.point708* noalias sret(%struct.point708) align 4 %0, %struct.point708* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point708* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point708* %0 to i8*
  %6 = bitcast %struct.point708* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo709(%struct.point709* noalias sret(%struct.point709) align 4 %0, %struct.point709* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point709* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point709* %0 to i8*
  %6 = bitcast %struct.point709* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo710([2 x i64] %0) #0 {
  %2 = alloca %struct.point710, align 4
  %3 = alloca %struct.point710, align 4
  %4 = bitcast %struct.point710* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point710* %2 to i8*
  %6 = bitcast %struct.point710* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point710* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo711(%struct.point711* noalias sret(%struct.point711) align 4 %0, %struct.point711* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point711* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point711* %0 to i8*
  %6 = bitcast %struct.point711* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo712(%struct.point712* noalias sret(%struct.point712) align 4 %0, %struct.point712* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point712* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point712* %0 to i8*
  %6 = bitcast %struct.point712* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo713(%struct.point713* noalias sret(%struct.point713) align 4 %0, %struct.point713* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point713* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point713* %0 to i8*
  %6 = bitcast %struct.point713* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo714(%struct.point714* noalias sret(%struct.point714) align 4 %0, %struct.point714* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point714* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point714* %0 to i8*
  %6 = bitcast %struct.point714* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo715([2 x i64] %0) #0 {
  %2 = alloca %struct.point715, align 4
  %3 = alloca %struct.point715, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point715* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point715* %2 to i8*
  %9 = bitcast %struct.point715* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point715* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo716([2 x i64] %0) #0 {
  %2 = alloca %struct.point716, align 4
  %3 = alloca %struct.point716, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point716* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point716* %2 to i8*
  %9 = bitcast %struct.point716* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point716* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo717([2 x i64] %0) #0 {
  %2 = alloca %struct.point717, align 4
  %3 = alloca %struct.point717, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point717* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point717* %2 to i8*
  %9 = bitcast %struct.point717* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point717* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo718([2 x i64] %0) #0 {
  %2 = alloca %struct.point718, align 4
  %3 = alloca %struct.point718, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point718* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point718* %2 to i8*
  %9 = bitcast %struct.point718* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point718* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo719([2 x i64] %0) #0 {
  %2 = alloca %struct.point719, align 4
  %3 = alloca %struct.point719, align 4
  %4 = bitcast %struct.point719* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point719* %2 to i8*
  %6 = bitcast %struct.point719* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point719* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo720([2 x i64] %0) #0 {
  %2 = alloca %struct.point720, align 4
  %3 = alloca %struct.point720, align 4
  %4 = bitcast %struct.point720* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point720* %2 to i8*
  %6 = bitcast %struct.point720* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point720* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo721([2 x i64] %0) #0 {
  %2 = alloca %struct.point721, align 4
  %3 = alloca %struct.point721, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point721* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point721* %2 to i8*
  %9 = bitcast %struct.point721* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point721* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo722([2 x i64] %0) #0 {
  %2 = alloca %struct.point722, align 4
  %3 = alloca %struct.point722, align 4
  %4 = bitcast %struct.point722* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point722* %2 to i8*
  %6 = bitcast %struct.point722* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point722* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo723([2 x i64] %0) #0 {
  %2 = alloca %struct.point723, align 4
  %3 = alloca %struct.point723, align 4
  %4 = bitcast %struct.point723* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point723* %2 to i8*
  %6 = bitcast %struct.point723* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point723* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo724([2 x i64] %0) #0 {
  %2 = alloca %struct.point724, align 4
  %3 = alloca %struct.point724, align 4
  %4 = bitcast %struct.point724* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point724* %2 to i8*
  %6 = bitcast %struct.point724* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point724* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo725([2 x i64] %0) #0 {
  %2 = alloca %struct.point725, align 4
  %3 = alloca %struct.point725, align 4
  %4 = bitcast %struct.point725* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point725* %2 to i8*
  %6 = bitcast %struct.point725* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point725* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo726([2 x i64] %0) #0 {
  %2 = alloca %struct.point726, align 4
  %3 = alloca %struct.point726, align 4
  %4 = bitcast %struct.point726* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point726* %2 to i8*
  %6 = bitcast %struct.point726* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point726* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo727(%struct.point727* noalias sret(%struct.point727) align 4 %0, %struct.point727* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point727* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point727* %0 to i8*
  %6 = bitcast %struct.point727* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo728(%struct.point728* noalias sret(%struct.point728) align 4 %0, %struct.point728* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point728* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point728* %0 to i8*
  %6 = bitcast %struct.point728* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo729(%struct.point729* noalias sret(%struct.point729) align 4 %0, %struct.point729* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point729* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point729* %0 to i8*
  %6 = bitcast %struct.point729* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo730(%struct.point730* noalias sret(%struct.point730) align 4 %0, %struct.point730* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point730* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point730* %0 to i8*
  %6 = bitcast %struct.point730* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo731([2 x i64] %0) #0 {
  %2 = alloca %struct.point731, align 4
  %3 = alloca %struct.point731, align 4
  %4 = bitcast %struct.point731* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point731* %2 to i8*
  %6 = bitcast %struct.point731* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point731* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo732(%struct.point732* noalias sret(%struct.point732) align 4 %0, %struct.point732* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point732* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point732* %0 to i8*
  %6 = bitcast %struct.point732* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo733(%struct.point733* noalias sret(%struct.point733) align 4 %0, %struct.point733* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point733* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point733* %0 to i8*
  %6 = bitcast %struct.point733* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo734(%struct.point734* noalias sret(%struct.point734) align 4 %0, %struct.point734* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point734* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point734* %0 to i8*
  %6 = bitcast %struct.point734* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo735(%struct.point735* noalias sret(%struct.point735) align 4 %0, %struct.point735* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point735* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point735* %0 to i8*
  %6 = bitcast %struct.point735* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo736([2 x i64] %0) #0 {
  %2 = alloca %struct.point736, align 4
  %3 = alloca %struct.point736, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point736* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point736* %2 to i8*
  %9 = bitcast %struct.point736* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point736* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo737([2 x i64] %0) #0 {
  %2 = alloca %struct.point737, align 4
  %3 = alloca %struct.point737, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point737* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point737* %2 to i8*
  %9 = bitcast %struct.point737* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point737* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo738([2 x i64] %0) #0 {
  %2 = alloca %struct.point738, align 4
  %3 = alloca %struct.point738, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point738* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point738* %2 to i8*
  %9 = bitcast %struct.point738* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point738* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo739([2 x i64] %0) #0 {
  %2 = alloca %struct.point739, align 4
  %3 = alloca %struct.point739, align 4
  %4 = bitcast %struct.point739* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point739* %2 to i8*
  %6 = bitcast %struct.point739* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point739* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo740([2 x i64] %0) #0 {
  %2 = alloca %struct.point740, align 4
  %3 = alloca %struct.point740, align 4
  %4 = bitcast %struct.point740* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point740* %2 to i8*
  %6 = bitcast %struct.point740* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point740* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo741([2 x i64] %0) #0 {
  %2 = alloca %struct.point741, align 4
  %3 = alloca %struct.point741, align 4
  %4 = bitcast %struct.point741* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point741* %2 to i8*
  %6 = bitcast %struct.point741* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point741* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo742([2 x i64] %0) #0 {
  %2 = alloca %struct.point742, align 4
  %3 = alloca %struct.point742, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point742* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point742* %2 to i8*
  %9 = bitcast %struct.point742* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point742* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo743([2 x i64] %0) #0 {
  %2 = alloca %struct.point743, align 4
  %3 = alloca %struct.point743, align 4
  %4 = bitcast %struct.point743* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point743* %2 to i8*
  %6 = bitcast %struct.point743* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point743* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo744([2 x i64] %0) #0 {
  %2 = alloca %struct.point744, align 4
  %3 = alloca %struct.point744, align 4
  %4 = bitcast %struct.point744* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point744* %2 to i8*
  %6 = bitcast %struct.point744* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point744* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo745([2 x i64] %0) #0 {
  %2 = alloca %struct.point745, align 4
  %3 = alloca %struct.point745, align 4
  %4 = bitcast %struct.point745* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point745* %2 to i8*
  %6 = bitcast %struct.point745* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point745* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo746([2 x i64] %0) #0 {
  %2 = alloca %struct.point746, align 4
  %3 = alloca %struct.point746, align 4
  %4 = bitcast %struct.point746* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point746* %2 to i8*
  %6 = bitcast %struct.point746* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point746* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo747([2 x i64] %0) #0 {
  %2 = alloca %struct.point747, align 4
  %3 = alloca %struct.point747, align 4
  %4 = bitcast %struct.point747* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point747* %2 to i8*
  %6 = bitcast %struct.point747* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point747* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo748(%struct.point748* noalias sret(%struct.point748) align 4 %0, %struct.point748* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point748* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point748* %0 to i8*
  %6 = bitcast %struct.point748* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo749(%struct.point749* noalias sret(%struct.point749) align 4 %0, %struct.point749* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point749* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point749* %0 to i8*
  %6 = bitcast %struct.point749* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo750(%struct.point750* noalias sret(%struct.point750) align 4 %0, %struct.point750* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point750* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point750* %0 to i8*
  %6 = bitcast %struct.point750* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo751(%struct.point751* noalias sret(%struct.point751) align 4 %0, %struct.point751* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point751* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point751* %0 to i8*
  %6 = bitcast %struct.point751* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo752([2 x i64] %0) #0 {
  %2 = alloca %struct.point752, align 4
  %3 = alloca %struct.point752, align 4
  %4 = bitcast %struct.point752* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point752* %2 to i8*
  %6 = bitcast %struct.point752* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point752* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo753(%struct.point753* noalias sret(%struct.point753) align 4 %0, %struct.point753* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point753* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point753* %0 to i8*
  %6 = bitcast %struct.point753* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo754(%struct.point754* noalias sret(%struct.point754) align 4 %0, %struct.point754* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point754* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point754* %0 to i8*
  %6 = bitcast %struct.point754* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo755(%struct.point755* noalias sret(%struct.point755) align 4 %0, %struct.point755* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point755* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point755* %0 to i8*
  %6 = bitcast %struct.point755* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo756(%struct.point756* noalias sret(%struct.point756) align 4 %0, %struct.point756* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point756* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point756* %0 to i8*
  %6 = bitcast %struct.point756* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo757([2 x i64] %0) #0 {
  %2 = alloca %struct.point757, align 4
  %3 = alloca %struct.point757, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point757* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point757* %2 to i8*
  %9 = bitcast %struct.point757* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point757* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo758([2 x i64] %0) #0 {
  %2 = alloca %struct.point758, align 4
  %3 = alloca %struct.point758, align 4
  %4 = bitcast %struct.point758* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point758* %2 to i8*
  %6 = bitcast %struct.point758* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point758* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo759([2 x i64] %0) #0 {
  %2 = alloca %struct.point759, align 4
  %3 = alloca %struct.point759, align 4
  %4 = bitcast %struct.point759* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point759* %2 to i8*
  %6 = bitcast %struct.point759* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point759* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo760([2 x i64] %0) #0 {
  %2 = alloca %struct.point760, align 4
  %3 = alloca %struct.point760, align 4
  %4 = bitcast %struct.point760* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point760* %2 to i8*
  %6 = bitcast %struct.point760* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point760* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo761(%struct.point761* noalias sret(%struct.point761) align 4 %0, %struct.point761* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point761* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point761* %0 to i8*
  %6 = bitcast %struct.point761* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo762(%struct.point762* noalias sret(%struct.point762) align 4 %0, %struct.point762* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point762* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point762* %0 to i8*
  %6 = bitcast %struct.point762* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo763([2 x i64] %0) #0 {
  %2 = alloca %struct.point763, align 4
  %3 = alloca %struct.point763, align 4
  %4 = bitcast %struct.point763* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point763* %2 to i8*
  %6 = bitcast %struct.point763* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point763* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo764([2 x i64] %0) #0 {
  %2 = alloca %struct.point764, align 4
  %3 = alloca %struct.point764, align 4
  %4 = bitcast %struct.point764* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point764* %2 to i8*
  %6 = bitcast %struct.point764* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point764* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo765([2 x i64] %0) #0 {
  %2 = alloca %struct.point765, align 4
  %3 = alloca %struct.point765, align 4
  %4 = bitcast %struct.point765* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point765* %2 to i8*
  %6 = bitcast %struct.point765* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point765* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo766(%struct.point766* noalias sret(%struct.point766) align 4 %0, %struct.point766* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point766* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point766* %0 to i8*
  %6 = bitcast %struct.point766* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo767(%struct.point767* noalias sret(%struct.point767) align 4 %0, %struct.point767* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point767* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point767* %0 to i8*
  %6 = bitcast %struct.point767* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo768([2 x i64] %0) #0 {
  %2 = alloca %struct.point768, align 4
  %3 = alloca %struct.point768, align 4
  %4 = bitcast %struct.point768* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point768* %2 to i8*
  %6 = bitcast %struct.point768* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point768* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo769(%struct.point769* noalias sret(%struct.point769) align 4 %0, %struct.point769* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point769* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point769* %0 to i8*
  %6 = bitcast %struct.point769* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo770(%struct.point770* noalias sret(%struct.point770) align 4 %0, %struct.point770* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point770* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point770* %0 to i8*
  %6 = bitcast %struct.point770* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo771(%struct.point771* noalias sret(%struct.point771) align 4 %0, %struct.point771* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point771* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point771* %0 to i8*
  %6 = bitcast %struct.point771* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo772(%struct.point772* noalias sret(%struct.point772) align 4 %0, %struct.point772* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point772* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point772* %0 to i8*
  %6 = bitcast %struct.point772* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo773([2 x i64] %0) #0 {
  %2 = alloca %struct.point773, align 4
  %3 = alloca %struct.point773, align 4
  %4 = bitcast %struct.point773* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point773* %2 to i8*
  %6 = bitcast %struct.point773* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point773* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo774(%struct.point774* noalias sret(%struct.point774) align 4 %0, %struct.point774* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point774* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point774* %0 to i8*
  %6 = bitcast %struct.point774* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo775(%struct.point775* noalias sret(%struct.point775) align 4 %0, %struct.point775* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point775* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point775* %0 to i8*
  %6 = bitcast %struct.point775* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo776(%struct.point776* noalias sret(%struct.point776) align 4 %0, %struct.point776* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point776* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point776* %0 to i8*
  %6 = bitcast %struct.point776* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo777(%struct.point777* noalias sret(%struct.point777) align 4 %0, %struct.point777* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point777* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point777* %0 to i8*
  %6 = bitcast %struct.point777* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo778([2 x i64] %0) #0 {
  %2 = alloca %struct.point778, align 4
  %3 = alloca %struct.point778, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point778* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point778* %2 to i8*
  %9 = bitcast %struct.point778* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point778* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo779([2 x i64] %0) #0 {
  %2 = alloca %struct.point779, align 4
  %3 = alloca %struct.point779, align 4
  %4 = bitcast %struct.point779* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point779* %2 to i8*
  %6 = bitcast %struct.point779* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point779* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo780([2 x i64] %0) #0 {
  %2 = alloca %struct.point780, align 4
  %3 = alloca %struct.point780, align 4
  %4 = bitcast %struct.point780* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point780* %2 to i8*
  %6 = bitcast %struct.point780* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point780* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo781([2 x i64] %0) #0 {
  %2 = alloca %struct.point781, align 4
  %3 = alloca %struct.point781, align 4
  %4 = bitcast %struct.point781* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point781* %2 to i8*
  %6 = bitcast %struct.point781* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point781* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo782(%struct.point782* noalias sret(%struct.point782) align 4 %0, %struct.point782* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point782* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point782* %0 to i8*
  %6 = bitcast %struct.point782* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo783(%struct.point783* noalias sret(%struct.point783) align 4 %0, %struct.point783* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point783* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point783* %0 to i8*
  %6 = bitcast %struct.point783* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo784([2 x i64] %0) #0 {
  %2 = alloca %struct.point784, align 4
  %3 = alloca %struct.point784, align 4
  %4 = bitcast %struct.point784* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point784* %2 to i8*
  %6 = bitcast %struct.point784* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point784* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo785([2 x i64] %0) #0 {
  %2 = alloca %struct.point785, align 4
  %3 = alloca %struct.point785, align 4
  %4 = bitcast %struct.point785* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point785* %2 to i8*
  %6 = bitcast %struct.point785* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point785* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo786([2 x i64] %0) #0 {
  %2 = alloca %struct.point786, align 4
  %3 = alloca %struct.point786, align 4
  %4 = bitcast %struct.point786* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point786* %2 to i8*
  %6 = bitcast %struct.point786* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point786* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo787(%struct.point787* noalias sret(%struct.point787) align 4 %0, %struct.point787* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point787* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point787* %0 to i8*
  %6 = bitcast %struct.point787* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo788(%struct.point788* noalias sret(%struct.point788) align 4 %0, %struct.point788* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point788* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point788* %0 to i8*
  %6 = bitcast %struct.point788* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo789([2 x i64] %0) #0 {
  %2 = alloca %struct.point789, align 4
  %3 = alloca %struct.point789, align 4
  %4 = bitcast %struct.point789* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point789* %2 to i8*
  %6 = bitcast %struct.point789* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point789* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo790(%struct.point790* noalias sret(%struct.point790) align 4 %0, %struct.point790* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point790* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point790* %0 to i8*
  %6 = bitcast %struct.point790* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo791(%struct.point791* noalias sret(%struct.point791) align 4 %0, %struct.point791* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point791* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point791* %0 to i8*
  %6 = bitcast %struct.point791* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo792(%struct.point792* noalias sret(%struct.point792) align 4 %0, %struct.point792* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point792* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point792* %0 to i8*
  %6 = bitcast %struct.point792* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo793(%struct.point793* noalias sret(%struct.point793) align 4 %0, %struct.point793* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point793* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point793* %0 to i8*
  %6 = bitcast %struct.point793* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo794([2 x i64] %0) #0 {
  %2 = alloca %struct.point794, align 4
  %3 = alloca %struct.point794, align 4
  %4 = bitcast %struct.point794* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point794* %2 to i8*
  %6 = bitcast %struct.point794* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point794* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo795(%struct.point795* noalias sret(%struct.point795) align 4 %0, %struct.point795* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point795* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point795* %0 to i8*
  %6 = bitcast %struct.point795* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo796(%struct.point796* noalias sret(%struct.point796) align 4 %0, %struct.point796* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point796* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point796* %0 to i8*
  %6 = bitcast %struct.point796* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo797(%struct.point797* noalias sret(%struct.point797) align 4 %0, %struct.point797* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point797* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point797* %0 to i8*
  %6 = bitcast %struct.point797* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo798(%struct.point798* noalias sret(%struct.point798) align 4 %0, %struct.point798* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point798* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point798* %0 to i8*
  %6 = bitcast %struct.point798* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo799(i64 %0) #0 {
  %2 = alloca %struct.point799, align 4
  %3 = alloca %struct.point799, align 4
  %4 = bitcast %struct.point799* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point799* %2 to i8*
  %6 = bitcast %struct.point799* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point799* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo800(i64 %0) #0 {
  %2 = alloca %struct.point800, align 4
  %3 = alloca %struct.point800, align 4
  %4 = bitcast %struct.point800* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point800* %2 to i8*
  %6 = bitcast %struct.point800* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point800* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo801(i64 %0) #0 {
  %2 = alloca %struct.point801, align 4
  %3 = alloca %struct.point801, align 4
  %4 = bitcast %struct.point801* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point801* %2 to i8*
  %6 = bitcast %struct.point801* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point801* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo802([2 x i64] %0) #0 {
  %2 = alloca %struct.point802, align 4
  %3 = alloca %struct.point802, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point802* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point802* %2 to i8*
  %9 = bitcast %struct.point802* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point802* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo803([2 x i64] %0) #0 {
  %2 = alloca %struct.point803, align 4
  %3 = alloca %struct.point803, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point803* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point803* %2 to i8*
  %9 = bitcast %struct.point803* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point803* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo804([2 x i64] %0) #0 {
  %2 = alloca %struct.point804, align 4
  %3 = alloca %struct.point804, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point804* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point804* %2 to i8*
  %9 = bitcast %struct.point804* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point804* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo805(i64 %0) #0 {
  %2 = alloca %struct.point805, align 4
  %3 = alloca %struct.point805, align 4
  %4 = bitcast %struct.point805* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point805* %2 to i8*
  %6 = bitcast %struct.point805* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point805* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo806([2 x i64] %0) #0 {
  %2 = alloca %struct.point806, align 4
  %3 = alloca %struct.point806, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point806* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point806* %2 to i8*
  %9 = bitcast %struct.point806* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point806* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo807([2 x i64] %0) #0 {
  %2 = alloca %struct.point807, align 4
  %3 = alloca %struct.point807, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point807* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point807* %2 to i8*
  %9 = bitcast %struct.point807* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point807* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo808([2 x i64] %0) #0 {
  %2 = alloca %struct.point808, align 4
  %3 = alloca %struct.point808, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point808* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point808* %2 to i8*
  %9 = bitcast %struct.point808* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point808* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo809([2 x i64] %0) #0 {
  %2 = alloca %struct.point809, align 4
  %3 = alloca %struct.point809, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point809* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point809* %2 to i8*
  %9 = bitcast %struct.point809* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point809* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo810([2 x i64] %0) #0 {
  %2 = alloca %struct.point810, align 4
  %3 = alloca %struct.point810, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point810* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point810* %2 to i8*
  %9 = bitcast %struct.point810* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point810* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo811([2 x i64] %0) #0 {
  %2 = alloca %struct.point811, align 4
  %3 = alloca %struct.point811, align 4
  %4 = bitcast %struct.point811* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point811* %2 to i8*
  %6 = bitcast %struct.point811* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point811* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo812([2 x i64] %0) #0 {
  %2 = alloca %struct.point812, align 4
  %3 = alloca %struct.point812, align 4
  %4 = bitcast %struct.point812* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point812* %2 to i8*
  %6 = bitcast %struct.point812* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point812* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo813([2 x i64] %0) #0 {
  %2 = alloca %struct.point813, align 4
  %3 = alloca %struct.point813, align 4
  %4 = bitcast %struct.point813* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point813* %2 to i8*
  %6 = bitcast %struct.point813* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point813* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo814([2 x i64] %0) #0 {
  %2 = alloca %struct.point814, align 4
  %3 = alloca %struct.point814, align 4
  %4 = bitcast %struct.point814* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point814* %2 to i8*
  %6 = bitcast %struct.point814* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point814* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo815([2 x i64] %0) #0 {
  %2 = alloca %struct.point815, align 4
  %3 = alloca %struct.point815, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point815* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point815* %2 to i8*
  %9 = bitcast %struct.point815* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point815* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo816([2 x i64] %0) #0 {
  %2 = alloca %struct.point816, align 4
  %3 = alloca %struct.point816, align 4
  %4 = bitcast %struct.point816* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point816* %2 to i8*
  %6 = bitcast %struct.point816* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point816* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo817([2 x i64] %0) #0 {
  %2 = alloca %struct.point817, align 4
  %3 = alloca %struct.point817, align 4
  %4 = bitcast %struct.point817* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point817* %2 to i8*
  %6 = bitcast %struct.point817* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point817* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo818([2 x i64] %0) #0 {
  %2 = alloca %struct.point818, align 4
  %3 = alloca %struct.point818, align 4
  %4 = bitcast %struct.point818* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point818* %2 to i8*
  %6 = bitcast %struct.point818* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point818* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo819([2 x i64] %0) #0 {
  %2 = alloca %struct.point819, align 4
  %3 = alloca %struct.point819, align 4
  %4 = bitcast %struct.point819* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point819* %2 to i8*
  %6 = bitcast %struct.point819* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point819* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo820(i64 %0) #0 {
  %2 = alloca %struct.point820, align 4
  %3 = alloca %struct.point820, align 4
  %4 = bitcast %struct.point820* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point820* %2 to i8*
  %6 = bitcast %struct.point820* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point820* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo821([2 x i64] %0) #0 {
  %2 = alloca %struct.point821, align 4
  %3 = alloca %struct.point821, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point821* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point821* %2 to i8*
  %9 = bitcast %struct.point821* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point821* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo822([2 x i64] %0) #0 {
  %2 = alloca %struct.point822, align 4
  %3 = alloca %struct.point822, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point822* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point822* %2 to i8*
  %9 = bitcast %struct.point822* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point822* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo823([2 x i64] %0) #0 {
  %2 = alloca %struct.point823, align 4
  %3 = alloca %struct.point823, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point823* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point823* %2 to i8*
  %9 = bitcast %struct.point823* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point823* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo824([2 x i64] %0) #0 {
  %2 = alloca %struct.point824, align 4
  %3 = alloca %struct.point824, align 4
  %4 = bitcast %struct.point824* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point824* %2 to i8*
  %6 = bitcast %struct.point824* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point824* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo825([2 x i64] %0) #0 {
  %2 = alloca %struct.point825, align 4
  %3 = alloca %struct.point825, align 4
  %4 = bitcast %struct.point825* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point825* %2 to i8*
  %6 = bitcast %struct.point825* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point825* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo826([2 x i64] %0) #0 {
  %2 = alloca %struct.point826, align 4
  %3 = alloca %struct.point826, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point826* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point826* %2 to i8*
  %9 = bitcast %struct.point826* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point826* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo827([2 x i64] %0) #0 {
  %2 = alloca %struct.point827, align 4
  %3 = alloca %struct.point827, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point827* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point827* %2 to i8*
  %9 = bitcast %struct.point827* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point827* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo828([2 x i64] %0) #0 {
  %2 = alloca %struct.point828, align 4
  %3 = alloca %struct.point828, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point828* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point828* %2 to i8*
  %9 = bitcast %struct.point828* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point828* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo829([2 x i64] %0) #0 {
  %2 = alloca %struct.point829, align 4
  %3 = alloca %struct.point829, align 4
  %4 = bitcast %struct.point829* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point829* %2 to i8*
  %6 = bitcast %struct.point829* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point829* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo830([2 x i64] %0) #0 {
  %2 = alloca %struct.point830, align 4
  %3 = alloca %struct.point830, align 4
  %4 = bitcast %struct.point830* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point830* %2 to i8*
  %6 = bitcast %struct.point830* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point830* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo831([2 x i64] %0) #0 {
  %2 = alloca %struct.point831, align 4
  %3 = alloca %struct.point831, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point831* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point831* %2 to i8*
  %9 = bitcast %struct.point831* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point831* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo832([2 x i64] %0) #0 {
  %2 = alloca %struct.point832, align 4
  %3 = alloca %struct.point832, align 4
  %4 = bitcast %struct.point832* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point832* %2 to i8*
  %6 = bitcast %struct.point832* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point832* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo833([2 x i64] %0) #0 {
  %2 = alloca %struct.point833, align 4
  %3 = alloca %struct.point833, align 4
  %4 = bitcast %struct.point833* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point833* %2 to i8*
  %6 = bitcast %struct.point833* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point833* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo834([2 x i64] %0) #0 {
  %2 = alloca %struct.point834, align 4
  %3 = alloca %struct.point834, align 4
  %4 = bitcast %struct.point834* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point834* %2 to i8*
  %6 = bitcast %struct.point834* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point834* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo835([2 x i64] %0) #0 {
  %2 = alloca %struct.point835, align 4
  %3 = alloca %struct.point835, align 4
  %4 = bitcast %struct.point835* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point835* %2 to i8*
  %6 = bitcast %struct.point835* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point835* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo836([2 x i64] %0) #0 {
  %2 = alloca %struct.point836, align 4
  %3 = alloca %struct.point836, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point836* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point836* %2 to i8*
  %9 = bitcast %struct.point836* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point836* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo837([2 x i64] %0) #0 {
  %2 = alloca %struct.point837, align 4
  %3 = alloca %struct.point837, align 4
  %4 = bitcast %struct.point837* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point837* %2 to i8*
  %6 = bitcast %struct.point837* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point837* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo838([2 x i64] %0) #0 {
  %2 = alloca %struct.point838, align 4
  %3 = alloca %struct.point838, align 4
  %4 = bitcast %struct.point838* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point838* %2 to i8*
  %6 = bitcast %struct.point838* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point838* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo839([2 x i64] %0) #0 {
  %2 = alloca %struct.point839, align 4
  %3 = alloca %struct.point839, align 4
  %4 = bitcast %struct.point839* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point839* %2 to i8*
  %6 = bitcast %struct.point839* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point839* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo840([2 x i64] %0) #0 {
  %2 = alloca %struct.point840, align 4
  %3 = alloca %struct.point840, align 4
  %4 = bitcast %struct.point840* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point840* %2 to i8*
  %6 = bitcast %struct.point840* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point840* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo841([2 x i64] %0) #0 {
  %2 = alloca %struct.point841, align 4
  %3 = alloca %struct.point841, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point841* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point841* %2 to i8*
  %9 = bitcast %struct.point841* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point841* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo842([2 x i64] %0) #0 {
  %2 = alloca %struct.point842, align 4
  %3 = alloca %struct.point842, align 4
  %4 = bitcast %struct.point842* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point842* %2 to i8*
  %6 = bitcast %struct.point842* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point842* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo843([2 x i64] %0) #0 {
  %2 = alloca %struct.point843, align 4
  %3 = alloca %struct.point843, align 4
  %4 = bitcast %struct.point843* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point843* %2 to i8*
  %6 = bitcast %struct.point843* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point843* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo844([2 x i64] %0) #0 {
  %2 = alloca %struct.point844, align 4
  %3 = alloca %struct.point844, align 4
  %4 = bitcast %struct.point844* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point844* %2 to i8*
  %6 = bitcast %struct.point844* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point844* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo845(%struct.point845* noalias sret(%struct.point845) align 4 %0, %struct.point845* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point845* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point845* %0 to i8*
  %6 = bitcast %struct.point845* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo846(%struct.point846* noalias sret(%struct.point846) align 4 %0, %struct.point846* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point846* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point846* %0 to i8*
  %6 = bitcast %struct.point846* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo847([2 x i64] %0) #0 {
  %2 = alloca %struct.point847, align 4
  %3 = alloca %struct.point847, align 4
  %4 = bitcast %struct.point847* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point847* %2 to i8*
  %6 = bitcast %struct.point847* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point847* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo848([2 x i64] %0) #0 {
  %2 = alloca %struct.point848, align 4
  %3 = alloca %struct.point848, align 4
  %4 = bitcast %struct.point848* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point848* %2 to i8*
  %6 = bitcast %struct.point848* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point848* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo849([2 x i64] %0) #0 {
  %2 = alloca %struct.point849, align 4
  %3 = alloca %struct.point849, align 4
  %4 = bitcast %struct.point849* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point849* %2 to i8*
  %6 = bitcast %struct.point849* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point849* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo850(%struct.point850* noalias sret(%struct.point850) align 4 %0, %struct.point850* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point850* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point850* %0 to i8*
  %6 = bitcast %struct.point850* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo851(%struct.point851* noalias sret(%struct.point851) align 4 %0, %struct.point851* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point851* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point851* %0 to i8*
  %6 = bitcast %struct.point851* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo852([2 x i64] %0) #0 {
  %2 = alloca %struct.point852, align 4
  %3 = alloca %struct.point852, align 4
  %4 = bitcast %struct.point852* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point852* %2 to i8*
  %6 = bitcast %struct.point852* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point852* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo853(%struct.point853* noalias sret(%struct.point853) align 4 %0, %struct.point853* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point853* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point853* %0 to i8*
  %6 = bitcast %struct.point853* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo854(%struct.point854* noalias sret(%struct.point854) align 4 %0, %struct.point854* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point854* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point854* %0 to i8*
  %6 = bitcast %struct.point854* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo855(%struct.point855* noalias sret(%struct.point855) align 4 %0, %struct.point855* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point855* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point855* %0 to i8*
  %6 = bitcast %struct.point855* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo856(%struct.point856* noalias sret(%struct.point856) align 4 %0, %struct.point856* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point856* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point856* %0 to i8*
  %6 = bitcast %struct.point856* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo857([2 x i64] %0) #0 {
  %2 = alloca %struct.point857, align 4
  %3 = alloca %struct.point857, align 4
  %4 = bitcast %struct.point857* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point857* %2 to i8*
  %6 = bitcast %struct.point857* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point857* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo858(%struct.point858* noalias sret(%struct.point858) align 4 %0, %struct.point858* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point858* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point858* %0 to i8*
  %6 = bitcast %struct.point858* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo859(%struct.point859* noalias sret(%struct.point859) align 4 %0, %struct.point859* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point859* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point859* %0 to i8*
  %6 = bitcast %struct.point859* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo860(%struct.point860* noalias sret(%struct.point860) align 4 %0, %struct.point860* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point860* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point860* %0 to i8*
  %6 = bitcast %struct.point860* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo861(%struct.point861* noalias sret(%struct.point861) align 4 %0, %struct.point861* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point861* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point861* %0 to i8*
  %6 = bitcast %struct.point861* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo862([2 x i64] %0) #0 {
  %2 = alloca %struct.point862, align 4
  %3 = alloca %struct.point862, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point862* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point862* %2 to i8*
  %9 = bitcast %struct.point862* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point862* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo863([2 x i64] %0) #0 {
  %2 = alloca %struct.point863, align 4
  %3 = alloca %struct.point863, align 4
  %4 = bitcast %struct.point863* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point863* %2 to i8*
  %6 = bitcast %struct.point863* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point863* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo864([2 x i64] %0) #0 {
  %2 = alloca %struct.point864, align 4
  %3 = alloca %struct.point864, align 4
  %4 = bitcast %struct.point864* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point864* %2 to i8*
  %6 = bitcast %struct.point864* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point864* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo865([2 x i64] %0) #0 {
  %2 = alloca %struct.point865, align 4
  %3 = alloca %struct.point865, align 4
  %4 = bitcast %struct.point865* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point865* %2 to i8*
  %6 = bitcast %struct.point865* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point865* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo866(%struct.point866* noalias sret(%struct.point866) align 4 %0, %struct.point866* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point866* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point866* %0 to i8*
  %6 = bitcast %struct.point866* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo867(%struct.point867* noalias sret(%struct.point867) align 4 %0, %struct.point867* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point867* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point867* %0 to i8*
  %6 = bitcast %struct.point867* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo868([2 x i64] %0) #0 {
  %2 = alloca %struct.point868, align 4
  %3 = alloca %struct.point868, align 4
  %4 = bitcast %struct.point868* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point868* %2 to i8*
  %6 = bitcast %struct.point868* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point868* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo869([2 x i64] %0) #0 {
  %2 = alloca %struct.point869, align 4
  %3 = alloca %struct.point869, align 4
  %4 = bitcast %struct.point869* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point869* %2 to i8*
  %6 = bitcast %struct.point869* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point869* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo870([2 x i64] %0) #0 {
  %2 = alloca %struct.point870, align 4
  %3 = alloca %struct.point870, align 4
  %4 = bitcast %struct.point870* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point870* %2 to i8*
  %6 = bitcast %struct.point870* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point870* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo871(%struct.point871* noalias sret(%struct.point871) align 4 %0, %struct.point871* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point871* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point871* %0 to i8*
  %6 = bitcast %struct.point871* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo872(%struct.point872* noalias sret(%struct.point872) align 4 %0, %struct.point872* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point872* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point872* %0 to i8*
  %6 = bitcast %struct.point872* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo873([2 x i64] %0) #0 {
  %2 = alloca %struct.point873, align 4
  %3 = alloca %struct.point873, align 4
  %4 = bitcast %struct.point873* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point873* %2 to i8*
  %6 = bitcast %struct.point873* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point873* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo874(%struct.point874* noalias sret(%struct.point874) align 4 %0, %struct.point874* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point874* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point874* %0 to i8*
  %6 = bitcast %struct.point874* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo875(%struct.point875* noalias sret(%struct.point875) align 4 %0, %struct.point875* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point875* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point875* %0 to i8*
  %6 = bitcast %struct.point875* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo876(%struct.point876* noalias sret(%struct.point876) align 4 %0, %struct.point876* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point876* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point876* %0 to i8*
  %6 = bitcast %struct.point876* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo877(%struct.point877* noalias sret(%struct.point877) align 4 %0, %struct.point877* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point877* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point877* %0 to i8*
  %6 = bitcast %struct.point877* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo878([2 x i64] %0) #0 {
  %2 = alloca %struct.point878, align 4
  %3 = alloca %struct.point878, align 4
  %4 = bitcast %struct.point878* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point878* %2 to i8*
  %6 = bitcast %struct.point878* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point878* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo879(%struct.point879* noalias sret(%struct.point879) align 4 %0, %struct.point879* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point879* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point879* %0 to i8*
  %6 = bitcast %struct.point879* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo880(%struct.point880* noalias sret(%struct.point880) align 4 %0, %struct.point880* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point880* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point880* %0 to i8*
  %6 = bitcast %struct.point880* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo881(%struct.point881* noalias sret(%struct.point881) align 4 %0, %struct.point881* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point881* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point881* %0 to i8*
  %6 = bitcast %struct.point881* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo882(%struct.point882* noalias sret(%struct.point882) align 4 %0, %struct.point882* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point882* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point882* %0 to i8*
  %6 = bitcast %struct.point882* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo883(i64 %0) #0 {
  %2 = alloca %struct.point883, align 4
  %3 = alloca %struct.point883, align 4
  %4 = bitcast %struct.point883* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point883* %2 to i8*
  %6 = bitcast %struct.point883* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point883* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo884(i64 %0) #0 {
  %2 = alloca %struct.point884, align 4
  %3 = alloca %struct.point884, align 4
  %4 = bitcast %struct.point884* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point884* %2 to i8*
  %6 = bitcast %struct.point884* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point884* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo885([2 x i64] %0) #0 {
  %2 = alloca %struct.point885, align 4
  %3 = alloca %struct.point885, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point885* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point885* %2 to i8*
  %9 = bitcast %struct.point885* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point885* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo886([2 x i64] %0) #0 {
  %2 = alloca %struct.point886, align 4
  %3 = alloca %struct.point886, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point886* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point886* %2 to i8*
  %9 = bitcast %struct.point886* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point886* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo887([2 x i64] %0) #0 {
  %2 = alloca %struct.point887, align 4
  %3 = alloca %struct.point887, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point887* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point887* %2 to i8*
  %9 = bitcast %struct.point887* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point887* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo888([2 x i64] %0) #0 {
  %2 = alloca %struct.point888, align 4
  %3 = alloca %struct.point888, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point888* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point888* %2 to i8*
  %9 = bitcast %struct.point888* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point888* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo889([2 x i64] %0) #0 {
  %2 = alloca %struct.point889, align 4
  %3 = alloca %struct.point889, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point889* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point889* %2 to i8*
  %9 = bitcast %struct.point889* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point889* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo890([2 x i64] %0) #0 {
  %2 = alloca %struct.point890, align 4
  %3 = alloca %struct.point890, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point890* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point890* %2 to i8*
  %9 = bitcast %struct.point890* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point890* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo891([2 x i64] %0) #0 {
  %2 = alloca %struct.point891, align 4
  %3 = alloca %struct.point891, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point891* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point891* %2 to i8*
  %9 = bitcast %struct.point891* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point891* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo892([2 x i64] %0) #0 {
  %2 = alloca %struct.point892, align 4
  %3 = alloca %struct.point892, align 4
  %4 = bitcast %struct.point892* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point892* %2 to i8*
  %6 = bitcast %struct.point892* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point892* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo893([2 x i64] %0) #0 {
  %2 = alloca %struct.point893, align 4
  %3 = alloca %struct.point893, align 4
  %4 = bitcast %struct.point893* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point893* %2 to i8*
  %6 = bitcast %struct.point893* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point893* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo894([2 x i64] %0) #0 {
  %2 = alloca %struct.point894, align 4
  %3 = alloca %struct.point894, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point894* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point894* %2 to i8*
  %9 = bitcast %struct.point894* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point894* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo895([2 x i64] %0) #0 {
  %2 = alloca %struct.point895, align 4
  %3 = alloca %struct.point895, align 4
  %4 = bitcast %struct.point895* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point895* %2 to i8*
  %6 = bitcast %struct.point895* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point895* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo896([2 x i64] %0) #0 {
  %2 = alloca %struct.point896, align 4
  %3 = alloca %struct.point896, align 4
  %4 = bitcast %struct.point896* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point896* %2 to i8*
  %6 = bitcast %struct.point896* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point896* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo897([2 x i64] %0) #0 {
  %2 = alloca %struct.point897, align 4
  %3 = alloca %struct.point897, align 4
  %4 = bitcast %struct.point897* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point897* %2 to i8*
  %6 = bitcast %struct.point897* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point897* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo898([2 x i64] %0) #0 {
  %2 = alloca %struct.point898, align 4
  %3 = alloca %struct.point898, align 4
  %4 = bitcast %struct.point898* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point898* %2 to i8*
  %6 = bitcast %struct.point898* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point898* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo899([2 x i64] %0) #0 {
  %2 = alloca %struct.point899, align 4
  %3 = alloca %struct.point899, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point899* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point899* %2 to i8*
  %9 = bitcast %struct.point899* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point899* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo900([2 x i64] %0) #0 {
  %2 = alloca %struct.point900, align 4
  %3 = alloca %struct.point900, align 4
  %4 = bitcast %struct.point900* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point900* %2 to i8*
  %6 = bitcast %struct.point900* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point900* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo901([2 x i64] %0) #0 {
  %2 = alloca %struct.point901, align 4
  %3 = alloca %struct.point901, align 4
  %4 = bitcast %struct.point901* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point901* %2 to i8*
  %6 = bitcast %struct.point901* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point901* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo902([2 x i64] %0) #0 {
  %2 = alloca %struct.point902, align 4
  %3 = alloca %struct.point902, align 4
  %4 = bitcast %struct.point902* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point902* %2 to i8*
  %6 = bitcast %struct.point902* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point902* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo903([2 x i64] %0) #0 {
  %2 = alloca %struct.point903, align 4
  %3 = alloca %struct.point903, align 4
  %4 = bitcast %struct.point903* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point903* %2 to i8*
  %6 = bitcast %struct.point903* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point903* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local i64 @demo904(i64 %0) #0 {
  %2 = alloca %struct.point904, align 4
  %3 = alloca %struct.point904, align 4
  %4 = bitcast %struct.point904* %3 to i64*
  store i64 %0, i64* %4, align 4
  %5 = bitcast %struct.point904* %2 to i8*
  %6 = bitcast %struct.point904* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 8, i1 false)
  %7 = bitcast %struct.point904* %2 to i64*
  %8 = load i64, i64* %7, align 4
  ret i64 %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo905([2 x i64] %0) #0 {
  %2 = alloca %struct.point905, align 4
  %3 = alloca %struct.point905, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point905* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point905* %2 to i8*
  %9 = bitcast %struct.point905* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point905* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo906([2 x i64] %0) #0 {
  %2 = alloca %struct.point906, align 4
  %3 = alloca %struct.point906, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point906* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point906* %2 to i8*
  %9 = bitcast %struct.point906* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point906* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo907([2 x i64] %0) #0 {
  %2 = alloca %struct.point907, align 4
  %3 = alloca %struct.point907, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point907* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point907* %2 to i8*
  %9 = bitcast %struct.point907* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point907* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo908([2 x i64] %0) #0 {
  %2 = alloca %struct.point908, align 4
  %3 = alloca %struct.point908, align 4
  %4 = bitcast %struct.point908* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point908* %2 to i8*
  %6 = bitcast %struct.point908* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point908* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo909([2 x i64] %0) #0 {
  %2 = alloca %struct.point909, align 4
  %3 = alloca %struct.point909, align 4
  %4 = bitcast %struct.point909* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point909* %2 to i8*
  %6 = bitcast %struct.point909* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point909* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo910([2 x i64] %0) #0 {
  %2 = alloca %struct.point910, align 4
  %3 = alloca %struct.point910, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point910* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point910* %2 to i8*
  %9 = bitcast %struct.point910* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point910* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo911([2 x i64] %0) #0 {
  %2 = alloca %struct.point911, align 4
  %3 = alloca %struct.point911, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point911* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point911* %2 to i8*
  %9 = bitcast %struct.point911* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point911* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo912([2 x i64] %0) #0 {
  %2 = alloca %struct.point912, align 4
  %3 = alloca %struct.point912, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point912* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point912* %2 to i8*
  %9 = bitcast %struct.point912* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point912* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo913([2 x i64] %0) #0 {
  %2 = alloca %struct.point913, align 4
  %3 = alloca %struct.point913, align 4
  %4 = bitcast %struct.point913* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point913* %2 to i8*
  %6 = bitcast %struct.point913* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point913* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo914([2 x i64] %0) #0 {
  %2 = alloca %struct.point914, align 4
  %3 = alloca %struct.point914, align 4
  %4 = bitcast %struct.point914* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point914* %2 to i8*
  %6 = bitcast %struct.point914* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point914* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo915([2 x i64] %0) #0 {
  %2 = alloca %struct.point915, align 4
  %3 = alloca %struct.point915, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point915* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point915* %2 to i8*
  %9 = bitcast %struct.point915* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point915* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo916([2 x i64] %0) #0 {
  %2 = alloca %struct.point916, align 4
  %3 = alloca %struct.point916, align 4
  %4 = bitcast %struct.point916* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point916* %2 to i8*
  %6 = bitcast %struct.point916* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point916* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo917([2 x i64] %0) #0 {
  %2 = alloca %struct.point917, align 4
  %3 = alloca %struct.point917, align 4
  %4 = bitcast %struct.point917* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point917* %2 to i8*
  %6 = bitcast %struct.point917* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point917* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo918([2 x i64] %0) #0 {
  %2 = alloca %struct.point918, align 4
  %3 = alloca %struct.point918, align 4
  %4 = bitcast %struct.point918* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point918* %2 to i8*
  %6 = bitcast %struct.point918* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point918* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo919([2 x i64] %0) #0 {
  %2 = alloca %struct.point919, align 4
  %3 = alloca %struct.point919, align 4
  %4 = bitcast %struct.point919* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point919* %2 to i8*
  %6 = bitcast %struct.point919* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point919* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo920([2 x i64] %0) #0 {
  %2 = alloca %struct.point920, align 4
  %3 = alloca %struct.point920, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point920* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point920* %2 to i8*
  %9 = bitcast %struct.point920* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point920* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo921([2 x i64] %0) #0 {
  %2 = alloca %struct.point921, align 4
  %3 = alloca %struct.point921, align 4
  %4 = bitcast %struct.point921* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point921* %2 to i8*
  %6 = bitcast %struct.point921* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point921* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo922([2 x i64] %0) #0 {
  %2 = alloca %struct.point922, align 4
  %3 = alloca %struct.point922, align 4
  %4 = bitcast %struct.point922* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point922* %2 to i8*
  %6 = bitcast %struct.point922* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point922* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo923([2 x i64] %0) #0 {
  %2 = alloca %struct.point923, align 4
  %3 = alloca %struct.point923, align 4
  %4 = bitcast %struct.point923* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point923* %2 to i8*
  %6 = bitcast %struct.point923* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point923* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo924([2 x i64] %0) #0 {
  %2 = alloca %struct.point924, align 4
  %3 = alloca %struct.point924, align 4
  %4 = bitcast %struct.point924* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point924* %2 to i8*
  %6 = bitcast %struct.point924* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point924* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo925([2 x i64] %0) #0 {
  %2 = alloca %struct.point925, align 4
  %3 = alloca %struct.point925, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point925* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point925* %2 to i8*
  %9 = bitcast %struct.point925* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point925* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo926([2 x i64] %0) #0 {
  %2 = alloca %struct.point926, align 4
  %3 = alloca %struct.point926, align 4
  %4 = bitcast %struct.point926* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point926* %2 to i8*
  %6 = bitcast %struct.point926* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point926* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo927([2 x i64] %0) #0 {
  %2 = alloca %struct.point927, align 4
  %3 = alloca %struct.point927, align 4
  %4 = bitcast %struct.point927* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point927* %2 to i8*
  %6 = bitcast %struct.point927* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point927* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo928([2 x i64] %0) #0 {
  %2 = alloca %struct.point928, align 4
  %3 = alloca %struct.point928, align 4
  %4 = bitcast %struct.point928* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point928* %2 to i8*
  %6 = bitcast %struct.point928* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point928* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo929(%struct.point929* noalias sret(%struct.point929) align 4 %0, %struct.point929* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point929* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point929* %0 to i8*
  %6 = bitcast %struct.point929* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo930(%struct.point930* noalias sret(%struct.point930) align 4 %0, %struct.point930* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point930* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point930* %0 to i8*
  %6 = bitcast %struct.point930* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo931([2 x i64] %0) #0 {
  %2 = alloca %struct.point931, align 4
  %3 = alloca %struct.point931, align 4
  %4 = bitcast %struct.point931* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point931* %2 to i8*
  %6 = bitcast %struct.point931* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point931* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo932([2 x i64] %0) #0 {
  %2 = alloca %struct.point932, align 4
  %3 = alloca %struct.point932, align 4
  %4 = bitcast %struct.point932* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point932* %2 to i8*
  %6 = bitcast %struct.point932* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point932* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo933([2 x i64] %0) #0 {
  %2 = alloca %struct.point933, align 4
  %3 = alloca %struct.point933, align 4
  %4 = bitcast %struct.point933* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point933* %2 to i8*
  %6 = bitcast %struct.point933* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point933* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo934(%struct.point934* noalias sret(%struct.point934) align 4 %0, %struct.point934* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point934* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point934* %0 to i8*
  %6 = bitcast %struct.point934* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo935(%struct.point935* noalias sret(%struct.point935) align 4 %0, %struct.point935* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point935* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point935* %0 to i8*
  %6 = bitcast %struct.point935* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo936([2 x i64] %0) #0 {
  %2 = alloca %struct.point936, align 4
  %3 = alloca %struct.point936, align 4
  %4 = bitcast %struct.point936* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point936* %2 to i8*
  %6 = bitcast %struct.point936* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point936* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo937(%struct.point937* noalias sret(%struct.point937) align 4 %0, %struct.point937* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point937* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point937* %0 to i8*
  %6 = bitcast %struct.point937* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo938(%struct.point938* noalias sret(%struct.point938) align 4 %0, %struct.point938* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point938* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point938* %0 to i8*
  %6 = bitcast %struct.point938* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo939(%struct.point939* noalias sret(%struct.point939) align 4 %0, %struct.point939* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point939* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point939* %0 to i8*
  %6 = bitcast %struct.point939* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo940(%struct.point940* noalias sret(%struct.point940) align 4 %0, %struct.point940* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point940* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point940* %0 to i8*
  %6 = bitcast %struct.point940* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo941([2 x i64] %0) #0 {
  %2 = alloca %struct.point941, align 4
  %3 = alloca %struct.point941, align 4
  %4 = bitcast %struct.point941* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point941* %2 to i8*
  %6 = bitcast %struct.point941* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point941* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo942(%struct.point942* noalias sret(%struct.point942) align 4 %0, %struct.point942* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point942* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point942* %0 to i8*
  %6 = bitcast %struct.point942* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo943(%struct.point943* noalias sret(%struct.point943) align 4 %0, %struct.point943* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point943* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point943* %0 to i8*
  %6 = bitcast %struct.point943* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo944(%struct.point944* noalias sret(%struct.point944) align 4 %0, %struct.point944* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point944* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point944* %0 to i8*
  %6 = bitcast %struct.point944* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo945(%struct.point945* noalias sret(%struct.point945) align 4 %0, %struct.point945* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point945* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point945* %0 to i8*
  %6 = bitcast %struct.point945* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo946([2 x i64] %0) #0 {
  %2 = alloca %struct.point946, align 4
  %3 = alloca %struct.point946, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point946* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point946* %2 to i8*
  %9 = bitcast %struct.point946* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point946* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo947([2 x i64] %0) #0 {
  %2 = alloca %struct.point947, align 4
  %3 = alloca %struct.point947, align 4
  %4 = bitcast %struct.point947* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point947* %2 to i8*
  %6 = bitcast %struct.point947* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point947* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo948([2 x i64] %0) #0 {
  %2 = alloca %struct.point948, align 4
  %3 = alloca %struct.point948, align 4
  %4 = bitcast %struct.point948* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point948* %2 to i8*
  %6 = bitcast %struct.point948* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point948* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo949([2 x i64] %0) #0 {
  %2 = alloca %struct.point949, align 4
  %3 = alloca %struct.point949, align 4
  %4 = bitcast %struct.point949* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point949* %2 to i8*
  %6 = bitcast %struct.point949* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point949* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo950(%struct.point950* noalias sret(%struct.point950) align 4 %0, %struct.point950* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point950* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point950* %0 to i8*
  %6 = bitcast %struct.point950* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo951(%struct.point951* noalias sret(%struct.point951) align 4 %0, %struct.point951* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point951* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point951* %0 to i8*
  %6 = bitcast %struct.point951* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo952([2 x i64] %0) #0 {
  %2 = alloca %struct.point952, align 4
  %3 = alloca %struct.point952, align 4
  %4 = bitcast %struct.point952* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point952* %2 to i8*
  %6 = bitcast %struct.point952* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point952* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo953([2 x i64] %0) #0 {
  %2 = alloca %struct.point953, align 4
  %3 = alloca %struct.point953, align 4
  %4 = bitcast %struct.point953* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point953* %2 to i8*
  %6 = bitcast %struct.point953* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point953* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo954([2 x i64] %0) #0 {
  %2 = alloca %struct.point954, align 4
  %3 = alloca %struct.point954, align 4
  %4 = bitcast %struct.point954* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point954* %2 to i8*
  %6 = bitcast %struct.point954* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point954* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo955(%struct.point955* noalias sret(%struct.point955) align 4 %0, %struct.point955* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point955* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point955* %0 to i8*
  %6 = bitcast %struct.point955* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo956(%struct.point956* noalias sret(%struct.point956) align 4 %0, %struct.point956* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point956* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point956* %0 to i8*
  %6 = bitcast %struct.point956* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo957([2 x i64] %0) #0 {
  %2 = alloca %struct.point957, align 4
  %3 = alloca %struct.point957, align 4
  %4 = bitcast %struct.point957* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point957* %2 to i8*
  %6 = bitcast %struct.point957* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point957* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo958(%struct.point958* noalias sret(%struct.point958) align 4 %0, %struct.point958* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point958* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point958* %0 to i8*
  %6 = bitcast %struct.point958* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo959(%struct.point959* noalias sret(%struct.point959) align 4 %0, %struct.point959* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point959* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point959* %0 to i8*
  %6 = bitcast %struct.point959* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo960(%struct.point960* noalias sret(%struct.point960) align 4 %0, %struct.point960* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point960* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point960* %0 to i8*
  %6 = bitcast %struct.point960* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo961(%struct.point961* noalias sret(%struct.point961) align 4 %0, %struct.point961* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point961* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point961* %0 to i8*
  %6 = bitcast %struct.point961* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo962([2 x i64] %0) #0 {
  %2 = alloca %struct.point962, align 4
  %3 = alloca %struct.point962, align 4
  %4 = bitcast %struct.point962* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point962* %2 to i8*
  %6 = bitcast %struct.point962* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point962* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo963(%struct.point963* noalias sret(%struct.point963) align 4 %0, %struct.point963* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point963* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point963* %0 to i8*
  %6 = bitcast %struct.point963* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo964(%struct.point964* noalias sret(%struct.point964) align 4 %0, %struct.point964* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point964* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point964* %0 to i8*
  %6 = bitcast %struct.point964* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo965(%struct.point965* noalias sret(%struct.point965) align 4 %0, %struct.point965* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point965* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point965* %0 to i8*
  %6 = bitcast %struct.point965* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo966(%struct.point966* noalias sret(%struct.point966) align 4 %0, %struct.point966* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point966* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point966* %0 to i8*
  %6 = bitcast %struct.point966* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo967([2 x i64] %0) #0 {
  %2 = alloca %struct.point967, align 4
  %3 = alloca %struct.point967, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point967* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point967* %2 to i8*
  %9 = bitcast %struct.point967* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point967* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo968([2 x i64] %0) #0 {
  %2 = alloca %struct.point968, align 4
  %3 = alloca %struct.point968, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point968* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point968* %2 to i8*
  %9 = bitcast %struct.point968* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point968* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo969([2 x i64] %0) #0 {
  %2 = alloca %struct.point969, align 4
  %3 = alloca %struct.point969, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point969* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point969* %2 to i8*
  %9 = bitcast %struct.point969* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point969* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo970([2 x i64] %0) #0 {
  %2 = alloca %struct.point970, align 4
  %3 = alloca %struct.point970, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point970* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point970* %2 to i8*
  %9 = bitcast %struct.point970* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point970* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo971([2 x i64] %0) #0 {
  %2 = alloca %struct.point971, align 4
  %3 = alloca %struct.point971, align 4
  %4 = bitcast %struct.point971* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point971* %2 to i8*
  %6 = bitcast %struct.point971* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point971* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo972([2 x i64] %0) #0 {
  %2 = alloca %struct.point972, align 4
  %3 = alloca %struct.point972, align 4
  %4 = bitcast %struct.point972* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point972* %2 to i8*
  %6 = bitcast %struct.point972* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point972* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo973([2 x i64] %0) #0 {
  %2 = alloca %struct.point973, align 4
  %3 = alloca %struct.point973, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point973* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point973* %2 to i8*
  %9 = bitcast %struct.point973* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point973* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo974([2 x i64] %0) #0 {
  %2 = alloca %struct.point974, align 4
  %3 = alloca %struct.point974, align 4
  %4 = bitcast %struct.point974* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point974* %2 to i8*
  %6 = bitcast %struct.point974* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point974* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo975([2 x i64] %0) #0 {
  %2 = alloca %struct.point975, align 4
  %3 = alloca %struct.point975, align 4
  %4 = bitcast %struct.point975* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point975* %2 to i8*
  %6 = bitcast %struct.point975* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point975* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo976([2 x i64] %0) #0 {
  %2 = alloca %struct.point976, align 4
  %3 = alloca %struct.point976, align 4
  %4 = bitcast %struct.point976* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point976* %2 to i8*
  %6 = bitcast %struct.point976* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point976* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo977([2 x i64] %0) #0 {
  %2 = alloca %struct.point977, align 4
  %3 = alloca %struct.point977, align 4
  %4 = bitcast %struct.point977* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point977* %2 to i8*
  %6 = bitcast %struct.point977* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point977* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo978([2 x i64] %0) #0 {
  %2 = alloca %struct.point978, align 4
  %3 = alloca %struct.point978, align 4
  %4 = bitcast %struct.point978* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point978* %2 to i8*
  %6 = bitcast %struct.point978* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point978* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo979(%struct.point979* noalias sret(%struct.point979) align 4 %0, %struct.point979* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point979* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point979* %0 to i8*
  %6 = bitcast %struct.point979* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo980(%struct.point980* noalias sret(%struct.point980) align 4 %0, %struct.point980* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point980* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point980* %0 to i8*
  %6 = bitcast %struct.point980* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo981(%struct.point981* noalias sret(%struct.point981) align 4 %0, %struct.point981* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point981* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point981* %0 to i8*
  %6 = bitcast %struct.point981* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo982(%struct.point982* noalias sret(%struct.point982) align 4 %0, %struct.point982* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point982* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point982* %0 to i8*
  %6 = bitcast %struct.point982* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo983([2 x i64] %0) #0 {
  %2 = alloca %struct.point983, align 4
  %3 = alloca %struct.point983, align 4
  %4 = bitcast %struct.point983* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point983* %2 to i8*
  %6 = bitcast %struct.point983* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point983* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo984(%struct.point984* noalias sret(%struct.point984) align 4 %0, %struct.point984* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point984* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point984* %0 to i8*
  %6 = bitcast %struct.point984* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo985(%struct.point985* noalias sret(%struct.point985) align 4 %0, %struct.point985* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point985* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point985* %0 to i8*
  %6 = bitcast %struct.point985* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo986(%struct.point986* noalias sret(%struct.point986) align 4 %0, %struct.point986* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point986* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point986* %0 to i8*
  %6 = bitcast %struct.point986* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo987(%struct.point987* noalias sret(%struct.point987) align 4 %0, %struct.point987* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point987* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point987* %0 to i8*
  %6 = bitcast %struct.point987* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo988([2 x i64] %0) #0 {
  %2 = alloca %struct.point988, align 4
  %3 = alloca %struct.point988, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point988* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point988* %2 to i8*
  %9 = bitcast %struct.point988* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point988* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo989([2 x i64] %0) #0 {
  %2 = alloca %struct.point989, align 4
  %3 = alloca %struct.point989, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point989* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point989* %2 to i8*
  %9 = bitcast %struct.point989* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point989* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo990([2 x i64] %0) #0 {
  %2 = alloca %struct.point990, align 4
  %3 = alloca %struct.point990, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point990* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point990* %2 to i8*
  %9 = bitcast %struct.point990* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point990* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo991([2 x i64] %0) #0 {
  %2 = alloca %struct.point991, align 4
  %3 = alloca %struct.point991, align 4
  %4 = bitcast %struct.point991* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point991* %2 to i8*
  %6 = bitcast %struct.point991* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point991* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo992([2 x i64] %0) #0 {
  %2 = alloca %struct.point992, align 4
  %3 = alloca %struct.point992, align 4
  %4 = bitcast %struct.point992* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point992* %2 to i8*
  %6 = bitcast %struct.point992* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point992* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo993([2 x i64] %0) #0 {
  %2 = alloca %struct.point993, align 4
  %3 = alloca %struct.point993, align 4
  %4 = bitcast %struct.point993* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point993* %2 to i8*
  %6 = bitcast %struct.point993* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point993* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo994([2 x i64] %0) #0 {
  %2 = alloca %struct.point994, align 4
  %3 = alloca %struct.point994, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point994* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point994* %2 to i8*
  %9 = bitcast %struct.point994* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point994* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo995([2 x i64] %0) #0 {
  %2 = alloca %struct.point995, align 4
  %3 = alloca %struct.point995, align 4
  %4 = bitcast %struct.point995* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point995* %2 to i8*
  %6 = bitcast %struct.point995* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point995* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo996([2 x i64] %0) #0 {
  %2 = alloca %struct.point996, align 4
  %3 = alloca %struct.point996, align 4
  %4 = bitcast %struct.point996* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point996* %2 to i8*
  %6 = bitcast %struct.point996* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point996* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo997([2 x i64] %0) #0 {
  %2 = alloca %struct.point997, align 4
  %3 = alloca %struct.point997, align 4
  %4 = bitcast %struct.point997* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point997* %2 to i8*
  %6 = bitcast %struct.point997* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point997* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo998([2 x i64] %0) #0 {
  %2 = alloca %struct.point998, align 4
  %3 = alloca %struct.point998, align 4
  %4 = bitcast %struct.point998* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point998* %2 to i8*
  %6 = bitcast %struct.point998* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point998* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo999([2 x i64] %0) #0 {
  %2 = alloca %struct.point999, align 4
  %3 = alloca %struct.point999, align 4
  %4 = bitcast %struct.point999* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point999* %2 to i8*
  %6 = bitcast %struct.point999* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point999* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1000(%struct.point1000* noalias sret(%struct.point1000) align 4 %0, %struct.point1000* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1000* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1000* %0 to i8*
  %6 = bitcast %struct.point1000* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1001(%struct.point1001* noalias sret(%struct.point1001) align 4 %0, %struct.point1001* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1001* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1001* %0 to i8*
  %6 = bitcast %struct.point1001* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1002(%struct.point1002* noalias sret(%struct.point1002) align 4 %0, %struct.point1002* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1002* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1002* %0 to i8*
  %6 = bitcast %struct.point1002* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1003(%struct.point1003* noalias sret(%struct.point1003) align 4 %0, %struct.point1003* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1003* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1003* %0 to i8*
  %6 = bitcast %struct.point1003* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1004([2 x i64] %0) #0 {
  %2 = alloca %struct.point1004, align 4
  %3 = alloca %struct.point1004, align 4
  %4 = bitcast %struct.point1004* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1004* %2 to i8*
  %6 = bitcast %struct.point1004* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1004* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1005(%struct.point1005* noalias sret(%struct.point1005) align 4 %0, %struct.point1005* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1005* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1005* %0 to i8*
  %6 = bitcast %struct.point1005* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1006(%struct.point1006* noalias sret(%struct.point1006) align 4 %0, %struct.point1006* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1006* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1006* %0 to i8*
  %6 = bitcast %struct.point1006* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1007(%struct.point1007* noalias sret(%struct.point1007) align 4 %0, %struct.point1007* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1007* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1007* %0 to i8*
  %6 = bitcast %struct.point1007* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1008(%struct.point1008* noalias sret(%struct.point1008) align 4 %0, %struct.point1008* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1008* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1008* %0 to i8*
  %6 = bitcast %struct.point1008* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1009([2 x i64] %0) #0 {
  %2 = alloca %struct.point1009, align 4
  %3 = alloca %struct.point1009, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point1009* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point1009* %2 to i8*
  %9 = bitcast %struct.point1009* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point1009* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1010([2 x i64] %0) #0 {
  %2 = alloca %struct.point1010, align 4
  %3 = alloca %struct.point1010, align 4
  %4 = bitcast %struct.point1010* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1010* %2 to i8*
  %6 = bitcast %struct.point1010* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1010* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1011([2 x i64] %0) #0 {
  %2 = alloca %struct.point1011, align 4
  %3 = alloca %struct.point1011, align 4
  %4 = bitcast %struct.point1011* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1011* %2 to i8*
  %6 = bitcast %struct.point1011* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1011* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1012([2 x i64] %0) #0 {
  %2 = alloca %struct.point1012, align 4
  %3 = alloca %struct.point1012, align 4
  %4 = bitcast %struct.point1012* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1012* %2 to i8*
  %6 = bitcast %struct.point1012* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1012* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1013(%struct.point1013* noalias sret(%struct.point1013) align 4 %0, %struct.point1013* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1013* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1013* %0 to i8*
  %6 = bitcast %struct.point1013* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1014(%struct.point1014* noalias sret(%struct.point1014) align 4 %0, %struct.point1014* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1014* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1014* %0 to i8*
  %6 = bitcast %struct.point1014* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1015([2 x i64] %0) #0 {
  %2 = alloca %struct.point1015, align 4
  %3 = alloca %struct.point1015, align 4
  %4 = bitcast %struct.point1015* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1015* %2 to i8*
  %6 = bitcast %struct.point1015* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1015* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1016([2 x i64] %0) #0 {
  %2 = alloca %struct.point1016, align 4
  %3 = alloca %struct.point1016, align 4
  %4 = bitcast %struct.point1016* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1016* %2 to i8*
  %6 = bitcast %struct.point1016* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1016* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1017([2 x i64] %0) #0 {
  %2 = alloca %struct.point1017, align 4
  %3 = alloca %struct.point1017, align 4
  %4 = bitcast %struct.point1017* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1017* %2 to i8*
  %6 = bitcast %struct.point1017* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1017* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1018(%struct.point1018* noalias sret(%struct.point1018) align 4 %0, %struct.point1018* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1018* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1018* %0 to i8*
  %6 = bitcast %struct.point1018* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1019(%struct.point1019* noalias sret(%struct.point1019) align 4 %0, %struct.point1019* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1019* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1019* %0 to i8*
  %6 = bitcast %struct.point1019* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1020([2 x i64] %0) #0 {
  %2 = alloca %struct.point1020, align 4
  %3 = alloca %struct.point1020, align 4
  %4 = bitcast %struct.point1020* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1020* %2 to i8*
  %6 = bitcast %struct.point1020* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1020* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1021(%struct.point1021* noalias sret(%struct.point1021) align 4 %0, %struct.point1021* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1021* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1021* %0 to i8*
  %6 = bitcast %struct.point1021* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1022(%struct.point1022* noalias sret(%struct.point1022) align 4 %0, %struct.point1022* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1022* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1022* %0 to i8*
  %6 = bitcast %struct.point1022* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1023(%struct.point1023* noalias sret(%struct.point1023) align 4 %0, %struct.point1023* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1023* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1023* %0 to i8*
  %6 = bitcast %struct.point1023* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1024(%struct.point1024* noalias sret(%struct.point1024) align 4 %0, %struct.point1024* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1024* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1024* %0 to i8*
  %6 = bitcast %struct.point1024* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1025([2 x i64] %0) #0 {
  %2 = alloca %struct.point1025, align 4
  %3 = alloca %struct.point1025, align 4
  %4 = bitcast %struct.point1025* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1025* %2 to i8*
  %6 = bitcast %struct.point1025* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1025* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1026(%struct.point1026* noalias sret(%struct.point1026) align 4 %0, %struct.point1026* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1026* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1026* %0 to i8*
  %6 = bitcast %struct.point1026* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1027(%struct.point1027* noalias sret(%struct.point1027) align 4 %0, %struct.point1027* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1027* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1027* %0 to i8*
  %6 = bitcast %struct.point1027* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1028(%struct.point1028* noalias sret(%struct.point1028) align 4 %0, %struct.point1028* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1028* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1028* %0 to i8*
  %6 = bitcast %struct.point1028* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1029(%struct.point1029* noalias sret(%struct.point1029) align 4 %0, %struct.point1029* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1029* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1029* %0 to i8*
  %6 = bitcast %struct.point1029* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1030([2 x i64] %0) #0 {
  %2 = alloca %struct.point1030, align 4
  %3 = alloca %struct.point1030, align 4
  %4 = alloca [2 x i64], align 8
  %5 = alloca [2 x i64], align 8
  store [2 x i64] %0, [2 x i64]* %4, align 8
  %6 = bitcast %struct.point1030* %3 to i8*
  %7 = bitcast [2 x i64]* %4 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %6, i8* align 8 %7, i64 12, i1 false)
  %8 = bitcast %struct.point1030* %2 to i8*
  %9 = bitcast %struct.point1030* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %8, i8* align 4 %9, i64 12, i1 false)
  %10 = bitcast [2 x i64]* %5 to i8*
  %11 = bitcast %struct.point1030* %2 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 8 %10, i8* align 4 %11, i64 12, i1 false)
  %12 = load [2 x i64], [2 x i64]* %5, align 8
  ret [2 x i64] %12
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1031([2 x i64] %0) #0 {
  %2 = alloca %struct.point1031, align 4
  %3 = alloca %struct.point1031, align 4
  %4 = bitcast %struct.point1031* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1031* %2 to i8*
  %6 = bitcast %struct.point1031* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1031* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1032([2 x i64] %0) #0 {
  %2 = alloca %struct.point1032, align 4
  %3 = alloca %struct.point1032, align 4
  %4 = bitcast %struct.point1032* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1032* %2 to i8*
  %6 = bitcast %struct.point1032* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1032* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1033([2 x i64] %0) #0 {
  %2 = alloca %struct.point1033, align 4
  %3 = alloca %struct.point1033, align 4
  %4 = bitcast %struct.point1033* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1033* %2 to i8*
  %6 = bitcast %struct.point1033* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1033* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1034(%struct.point1034* noalias sret(%struct.point1034) align 4 %0, %struct.point1034* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1034* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1034* %0 to i8*
  %6 = bitcast %struct.point1034* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1035(%struct.point1035* noalias sret(%struct.point1035) align 4 %0, %struct.point1035* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1035* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1035* %0 to i8*
  %6 = bitcast %struct.point1035* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1036([2 x i64] %0) #0 {
  %2 = alloca %struct.point1036, align 4
  %3 = alloca %struct.point1036, align 4
  %4 = bitcast %struct.point1036* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1036* %2 to i8*
  %6 = bitcast %struct.point1036* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1036* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1037([2 x i64] %0) #0 {
  %2 = alloca %struct.point1037, align 4
  %3 = alloca %struct.point1037, align 4
  %4 = bitcast %struct.point1037* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1037* %2 to i8*
  %6 = bitcast %struct.point1037* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1037* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1038([2 x i64] %0) #0 {
  %2 = alloca %struct.point1038, align 4
  %3 = alloca %struct.point1038, align 4
  %4 = bitcast %struct.point1038* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1038* %2 to i8*
  %6 = bitcast %struct.point1038* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1038* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1039(%struct.point1039* noalias sret(%struct.point1039) align 4 %0, %struct.point1039* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1039* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1039* %0 to i8*
  %6 = bitcast %struct.point1039* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1040(%struct.point1040* noalias sret(%struct.point1040) align 4 %0, %struct.point1040* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1040* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1040* %0 to i8*
  %6 = bitcast %struct.point1040* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1041([2 x i64] %0) #0 {
  %2 = alloca %struct.point1041, align 4
  %3 = alloca %struct.point1041, align 4
  %4 = bitcast %struct.point1041* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1041* %2 to i8*
  %6 = bitcast %struct.point1041* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1041* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1042(%struct.point1042* noalias sret(%struct.point1042) align 4 %0, %struct.point1042* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1042* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1042* %0 to i8*
  %6 = bitcast %struct.point1042* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1043(%struct.point1043* noalias sret(%struct.point1043) align 4 %0, %struct.point1043* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1043* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1043* %0 to i8*
  %6 = bitcast %struct.point1043* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1044(%struct.point1044* noalias sret(%struct.point1044) align 4 %0, %struct.point1044* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1044* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1044* %0 to i8*
  %6 = bitcast %struct.point1044* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1045(%struct.point1045* noalias sret(%struct.point1045) align 4 %0, %struct.point1045* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1045* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1045* %0 to i8*
  %6 = bitcast %struct.point1045* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo1046([2 x i64] %0) #0 {
  %2 = alloca %struct.point1046, align 4
  %3 = alloca %struct.point1046, align 4
  %4 = bitcast %struct.point1046* %3 to [2 x i64]*
  store [2 x i64] %0, [2 x i64]* %4, align 4
  %5 = bitcast %struct.point1046* %2 to i8*
  %6 = bitcast %struct.point1046* %3 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 16, i1 false)
  %7 = bitcast %struct.point1046* %2 to [2 x i64]*
  %8 = load [2 x i64], [2 x i64]* %7, align 4
  ret [2 x i64] %8
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1047(%struct.point1047* noalias sret(%struct.point1047) align 4 %0, %struct.point1047* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1047* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1047* %0 to i8*
  %6 = bitcast %struct.point1047* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1048(%struct.point1048* noalias sret(%struct.point1048) align 4 %0, %struct.point1048* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1048* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1048* %0 to i8*
  %6 = bitcast %struct.point1048* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1049(%struct.point1049* noalias sret(%struct.point1049) align 4 %0, %struct.point1049* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1049* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1049* %0 to i8*
  %6 = bitcast %struct.point1049* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1050(%struct.point1050* noalias sret(%struct.point1050) align 4 %0, %struct.point1050* noundef %1) #0 {
  %3 = alloca i8*, align 8
  %4 = bitcast %struct.point1050* %0 to i8*
  store i8* %4, i8** %3, align 8
  %5 = bitcast %struct.point1050* %0 to i8*
  %6 = bitcast %struct.point1050* %1 to i8*
  call void @llvm.memcpy.p0i8.p0i8.i64(i8* align 4 %5, i8* align 4 %6, i64 20, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "min-legal-vector-width"="0" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+64bit,+a,+c,+d,+f,+m,+relax,-save-restore" }
attributes #1 = { argmemonly nofree nounwind willreturn }

!llvm.module.flags = !{!0, !1, !2, !3}
!llvm.ident = !{!4}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 1, !"target-abi", !"lp64d"}
!2 = !{i32 7, !"frame-pointer", i32 2}
!3 = !{i32 1, !"SmallDataLimit", i32 8}
!4 = !{!"Apple clang version 14.0.3 (clang-1403.0.22.14.1)"}
