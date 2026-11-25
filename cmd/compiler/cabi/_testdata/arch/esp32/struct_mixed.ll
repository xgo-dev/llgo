; ModuleID = '../../wrap/struct_mixed.c'
source_filename = "../../wrap/struct_mixed.c"
target datalayout = "e-m:e-p:32:32-v1:8:8-i64:64-i128:128-n32"
target triple = "xtensa-esp-unknown-elf"

%struct.point1 = type { i8, i16 }
%struct.point2 = type { i8, i32 }
%struct.point3 = type { i8, i64 }
%struct.point4 = type { i8, float }
%struct.point5 = type { i8, double }
%struct.point6 = type { i8, ptr }
%struct.point7 = type { i16, i8 }
%struct.point8 = type { i16, i32 }
%struct.point9 = type { i16, i64 }
%struct.point10 = type { i16, float }
%struct.point11 = type { i16, double }
%struct.point12 = type { i16, ptr }
%struct.point13 = type { i32, i8 }
%struct.point14 = type { i32, i16 }
%struct.point15 = type { i32, i64 }
%struct.point16 = type { i32, float }
%struct.point17 = type { i32, double }
%struct.point18 = type { i32, ptr }
%struct.point19 = type { i64, i8 }
%struct.point20 = type { i64, i16 }
%struct.point21 = type { i64, i32 }
%struct.point22 = type { i64, float }
%struct.point23 = type { i64, double }
%struct.point24 = type { i64, ptr }
%struct.point25 = type { float, i8 }
%struct.point26 = type { float, i16 }
%struct.point27 = type { float, i32 }
%struct.point28 = type { float, i64 }
%struct.point29 = type { float, double }
%struct.point30 = type { float, ptr }
%struct.point31 = type { double, i8 }
%struct.point32 = type { double, i16 }
%struct.point33 = type { double, i32 }
%struct.point34 = type { double, i64 }
%struct.point35 = type { double, float }
%struct.point36 = type { double, ptr }
%struct.point37 = type { ptr, i8 }
%struct.point38 = type { ptr, i16 }
%struct.point39 = type { ptr, i32 }
%struct.point40 = type { ptr, i64 }
%struct.point41 = type { ptr, float }
%struct.point42 = type { ptr, double }
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
define dso_local void @demo1(ptr dead_on_unwind noalias writable sret(%struct.point1) align 2 %0, i32 %1) #0 {
  %3 = alloca %struct.point1, align 2
  store i32 %1, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %0, ptr align 2 %3, i32 4, i1 false)
  ret void
}

; Function Attrs: nocallback nofree nounwind willreturn memory(argmem: readwrite)
declare void @llvm.memcpy.p0.p0.i32(ptr noalias nocapture writeonly, ptr noalias nocapture readonly, i32, i1 immarg) #1

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo2([2 x i32] %0) #0 {
  %2 = alloca %struct.point2, align 4
  %3 = alloca %struct.point2, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo3([2 x i64] %0) #0 {
  %2 = alloca %struct.point3, align 8
  %3 = alloca %struct.point3, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo4([2 x i32] %0) #0 {
  %2 = alloca %struct.point4, align 4
  %3 = alloca %struct.point4, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo5([2 x i64] %0) #0 {
  %2 = alloca %struct.point5, align 8
  %3 = alloca %struct.point5, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo6([2 x i32] %0) #0 {
  %2 = alloca %struct.point6, align 4
  %3 = alloca %struct.point6, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo7(ptr dead_on_unwind noalias writable sret(%struct.point7) align 2 %0, i32 %1) #0 {
  %3 = alloca %struct.point7, align 2
  store i32 %1, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %0, ptr align 2 %3, i32 4, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo8([2 x i32] %0) #0 {
  %2 = alloca %struct.point8, align 4
  %3 = alloca %struct.point8, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo9([2 x i64] %0) #0 {
  %2 = alloca %struct.point9, align 8
  %3 = alloca %struct.point9, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo10([2 x i32] %0) #0 {
  %2 = alloca %struct.point10, align 4
  %3 = alloca %struct.point10, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo11([2 x i64] %0) #0 {
  %2 = alloca %struct.point11, align 8
  %3 = alloca %struct.point11, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo12([2 x i32] %0) #0 {
  %2 = alloca %struct.point12, align 4
  %3 = alloca %struct.point12, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo13([2 x i32] %0) #0 {
  %2 = alloca %struct.point13, align 4
  %3 = alloca %struct.point13, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo14([2 x i32] %0) #0 {
  %2 = alloca %struct.point14, align 4
  %3 = alloca %struct.point14, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo15([2 x i64] %0) #0 {
  %2 = alloca %struct.point15, align 8
  %3 = alloca %struct.point15, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo16([2 x i32] %0) #0 {
  %2 = alloca %struct.point16, align 4
  %3 = alloca %struct.point16, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo17([2 x i64] %0) #0 {
  %2 = alloca %struct.point17, align 8
  %3 = alloca %struct.point17, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo18([2 x i32] %0) #0 {
  %2 = alloca %struct.point18, align 4
  %3 = alloca %struct.point18, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo19([2 x i64] %0) #0 {
  %2 = alloca %struct.point19, align 8
  %3 = alloca %struct.point19, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo20([2 x i64] %0) #0 {
  %2 = alloca %struct.point20, align 8
  %3 = alloca %struct.point20, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo21([2 x i64] %0) #0 {
  %2 = alloca %struct.point21, align 8
  %3 = alloca %struct.point21, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo22([2 x i64] %0) #0 {
  %2 = alloca %struct.point22, align 8
  %3 = alloca %struct.point22, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo23([2 x i64] %0) #0 {
  %2 = alloca %struct.point23, align 8
  %3 = alloca %struct.point23, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo24([2 x i64] %0) #0 {
  %2 = alloca %struct.point24, align 8
  %3 = alloca %struct.point24, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo25([2 x i32] %0) #0 {
  %2 = alloca %struct.point25, align 4
  %3 = alloca %struct.point25, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo26([2 x i32] %0) #0 {
  %2 = alloca %struct.point26, align 4
  %3 = alloca %struct.point26, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo27([2 x i32] %0) #0 {
  %2 = alloca %struct.point27, align 4
  %3 = alloca %struct.point27, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo28([2 x i64] %0) #0 {
  %2 = alloca %struct.point28, align 8
  %3 = alloca %struct.point28, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo29([2 x i64] %0) #0 {
  %2 = alloca %struct.point29, align 8
  %3 = alloca %struct.point29, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo30([2 x i32] %0) #0 {
  %2 = alloca %struct.point30, align 4
  %3 = alloca %struct.point30, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo31([2 x i64] %0) #0 {
  %2 = alloca %struct.point31, align 8
  %3 = alloca %struct.point31, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo32([2 x i64] %0) #0 {
  %2 = alloca %struct.point32, align 8
  %3 = alloca %struct.point32, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo33([2 x i64] %0) #0 {
  %2 = alloca %struct.point33, align 8
  %3 = alloca %struct.point33, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo34([2 x i64] %0) #0 {
  %2 = alloca %struct.point34, align 8
  %3 = alloca %struct.point34, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo35([2 x i64] %0) #0 {
  %2 = alloca %struct.point35, align 8
  %3 = alloca %struct.point35, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo36([2 x i64] %0) #0 {
  %2 = alloca %struct.point36, align 8
  %3 = alloca %struct.point36, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo37([2 x i32] %0) #0 {
  %2 = alloca %struct.point37, align 4
  %3 = alloca %struct.point37, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo38([2 x i32] %0) #0 {
  %2 = alloca %struct.point38, align 4
  %3 = alloca %struct.point38, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo39([2 x i32] %0) #0 {
  %2 = alloca %struct.point39, align 4
  %3 = alloca %struct.point39, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo40([2 x i64] %0) #0 {
  %2 = alloca %struct.point40, align 8
  %3 = alloca %struct.point40, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo41([2 x i32] %0) #0 {
  %2 = alloca %struct.point41, align 4
  %3 = alloca %struct.point41, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i64] @demo42([2 x i64] %0) #0 {
  %2 = alloca %struct.point42, align 8
  %3 = alloca %struct.point42, align 8
  store [2 x i64] %0, ptr %3, align 8
  call void @llvm.memcpy.p0.p0.i32(ptr align 8 %2, ptr align 8 %3, i32 16, i1 false)
  %4 = load [2 x i64], ptr %2, align 8
  ret [2 x i64] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo43([2 x i32] %0) #0 {
  %2 = alloca %struct.point43, align 2
  %3 = alloca %struct.point43, align 2
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 6, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo44([2 x i32] %0) #0 {
  %2 = alloca %struct.point44, align 2
  %3 = alloca %struct.point44, align 2
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 6, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo45([2 x i32] %0) #0 {
  %2 = alloca %struct.point45, align 2
  %3 = alloca %struct.point45, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo46([2 x i32] %0) #0 {
  %2 = alloca %struct.point46, align 2
  %3 = alloca %struct.point46, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo47([3 x i32] %0) #0 {
  %2 = alloca %struct.point47, align 4
  %3 = alloca %struct.point47, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo48([3 x i32] %0) #0 {
  %2 = alloca %struct.point48, align 4
  %3 = alloca %struct.point48, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo49([2 x i32] %0) #0 {
  %2 = alloca %struct.point49, align 2
  %3 = alloca %struct.point49, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo50([3 x i32] %0) #0 {
  %2 = alloca %struct.point50, align 2
  %3 = alloca %struct.point50, align 2
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 10, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo51([3 x i32] %0) #0 {
  %2 = alloca %struct.point51, align 2
  %3 = alloca %struct.point51, align 2
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 10, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo52([3 x i32] %0) #0 {
  %2 = alloca %struct.point52, align 4
  %3 = alloca %struct.point52, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo53([3 x i32] %0) #0 {
  %2 = alloca %struct.point53, align 4
  %3 = alloca %struct.point53, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo54([3 x i32] %0) #0 {
  %2 = alloca %struct.point54, align 4
  %3 = alloca %struct.point54, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo55([4 x i32] %0) #0 {
  %2 = alloca %struct.point55, align 4
  %3 = alloca %struct.point55, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo56([4 x i32] %0) #0 {
  %2 = alloca %struct.point56, align 4
  %3 = alloca %struct.point56, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo57([4 x i32] %0) #0 {
  %2 = alloca %struct.point57, align 4
  %3 = alloca %struct.point57, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo58([4 x i32] %0) #0 {
  %2 = alloca %struct.point58, align 4
  %3 = alloca %struct.point58, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo59([3 x i32] %0) #0 {
  %2 = alloca %struct.point59, align 4
  %3 = alloca %struct.point59, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo60([4 x i32] %0) #0 {
  %2 = alloca %struct.point60, align 4
  %3 = alloca %struct.point60, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo61([4 x i32] %0) #0 {
  %2 = alloca %struct.point61, align 4
  %3 = alloca %struct.point61, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo62([4 x i32] %0) #0 {
  %2 = alloca %struct.point62, align 4
  %3 = alloca %struct.point62, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo63([4 x i32] %0) #0 {
  %2 = alloca %struct.point63, align 4
  %3 = alloca %struct.point63, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo64([2 x i32] %0) #0 {
  %2 = alloca %struct.point64, align 2
  %3 = alloca %struct.point64, align 2
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 6, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo65([2 x i32] %0) #0 {
  %2 = alloca %struct.point65, align 2
  %3 = alloca %struct.point65, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo66([2 x i32] %0) #0 {
  %2 = alloca %struct.point66, align 2
  %3 = alloca %struct.point66, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo67([3 x i32] %0) #0 {
  %2 = alloca %struct.point67, align 2
  %3 = alloca %struct.point67, align 2
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 10, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo68([3 x i32] %0) #0 {
  %2 = alloca %struct.point68, align 4
  %3 = alloca %struct.point68, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo69([3 x i32] %0) #0 {
  %2 = alloca %struct.point69, align 4
  %3 = alloca %struct.point69, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo70([2 x i32] %0) #0 {
  %2 = alloca %struct.point70, align 2
  %3 = alloca %struct.point70, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo71([3 x i32] %0) #0 {
  %2 = alloca %struct.point71, align 2
  %3 = alloca %struct.point71, align 2
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 10, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo72([3 x i32] %0) #0 {
  %2 = alloca %struct.point72, align 2
  %3 = alloca %struct.point72, align 2
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 10, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo73([3 x i32] %0) #0 {
  %2 = alloca %struct.point73, align 4
  %3 = alloca %struct.point73, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo74([3 x i32] %0) #0 {
  %2 = alloca %struct.point74, align 4
  %3 = alloca %struct.point74, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo75([3 x i32] %0) #0 {
  %2 = alloca %struct.point75, align 4
  %3 = alloca %struct.point75, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo76([4 x i32] %0) #0 {
  %2 = alloca %struct.point76, align 4
  %3 = alloca %struct.point76, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo77([4 x i32] %0) #0 {
  %2 = alloca %struct.point77, align 4
  %3 = alloca %struct.point77, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo78([4 x i32] %0) #0 {
  %2 = alloca %struct.point78, align 4
  %3 = alloca %struct.point78, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo79([4 x i32] %0) #0 {
  %2 = alloca %struct.point79, align 4
  %3 = alloca %struct.point79, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo80([3 x i32] %0) #0 {
  %2 = alloca %struct.point80, align 4
  %3 = alloca %struct.point80, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo81([4 x i32] %0) #0 {
  %2 = alloca %struct.point81, align 4
  %3 = alloca %struct.point81, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo82([4 x i32] %0) #0 {
  %2 = alloca %struct.point82, align 4
  %3 = alloca %struct.point82, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo83([4 x i32] %0) #0 {
  %2 = alloca %struct.point83, align 4
  %3 = alloca %struct.point83, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo84([4 x i32] %0) #0 {
  %2 = alloca %struct.point84, align 4
  %3 = alloca %struct.point84, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo85([2 x i32] %0) #0 {
  %2 = alloca %struct.point85, align 4
  %3 = alloca %struct.point85, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo86([3 x i32] %0) #0 {
  %2 = alloca %struct.point86, align 4
  %3 = alloca %struct.point86, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo87([3 x i32] %0) #0 {
  %2 = alloca %struct.point87, align 4
  %3 = alloca %struct.point87, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo88([3 x i32] %0) #0 {
  %2 = alloca %struct.point88, align 4
  %3 = alloca %struct.point88, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo89([4 x i32] %0) #0 {
  %2 = alloca %struct.point89, align 4
  %3 = alloca %struct.point89, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo90([4 x i32] %0) #0 {
  %2 = alloca %struct.point90, align 4
  %3 = alloca %struct.point90, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo91([3 x i32] %0) #0 {
  %2 = alloca %struct.point91, align 4
  %3 = alloca %struct.point91, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo92([3 x i32] %0) #0 {
  %2 = alloca %struct.point92, align 4
  %3 = alloca %struct.point92, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo93([3 x i32] %0) #0 {
  %2 = alloca %struct.point93, align 4
  %3 = alloca %struct.point93, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo94([4 x i32] %0) #0 {
  %2 = alloca %struct.point94, align 4
  %3 = alloca %struct.point94, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo95([4 x i32] %0) #0 {
  %2 = alloca %struct.point95, align 4
  %3 = alloca %struct.point95, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo96([3 x i32] %0) #0 {
  %2 = alloca %struct.point96, align 4
  %3 = alloca %struct.point96, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo97([4 x i32] %0) #0 {
  %2 = alloca %struct.point97, align 4
  %3 = alloca %struct.point97, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo98([4 x i32] %0) #0 {
  %2 = alloca %struct.point98, align 4
  %3 = alloca %struct.point98, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo99([4 x i32] %0) #0 {
  %2 = alloca %struct.point99, align 4
  %3 = alloca %struct.point99, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo100([4 x i32] %0) #0 {
  %2 = alloca %struct.point100, align 4
  %3 = alloca %struct.point100, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo101([3 x i32] %0) #0 {
  %2 = alloca %struct.point101, align 4
  %3 = alloca %struct.point101, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo102([4 x i32] %0) #0 {
  %2 = alloca %struct.point102, align 4
  %3 = alloca %struct.point102, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo103([4 x i32] %0) #0 {
  %2 = alloca %struct.point103, align 4
  %3 = alloca %struct.point103, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo104([4 x i32] %0) #0 {
  %2 = alloca %struct.point104, align 4
  %3 = alloca %struct.point104, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo105([4 x i32] %0) #0 {
  %2 = alloca %struct.point105, align 4
  %3 = alloca %struct.point105, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo106([2 x i32] %0) #0 {
  %2 = alloca %struct.point106, align 4
  %3 = alloca %struct.point106, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo107([3 x i32] %0) #0 {
  %2 = alloca %struct.point107, align 4
  %3 = alloca %struct.point107, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo108([3 x i32] %0) #0 {
  %2 = alloca %struct.point108, align 4
  %3 = alloca %struct.point108, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo109([3 x i32] %0) #0 {
  %2 = alloca %struct.point109, align 4
  %3 = alloca %struct.point109, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo110([4 x i32] %0) #0 {
  %2 = alloca %struct.point110, align 4
  %3 = alloca %struct.point110, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo111([4 x i32] %0) #0 {
  %2 = alloca %struct.point111, align 4
  %3 = alloca %struct.point111, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo112([3 x i32] %0) #0 {
  %2 = alloca %struct.point112, align 4
  %3 = alloca %struct.point112, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo113([3 x i32] %0) #0 {
  %2 = alloca %struct.point113, align 4
  %3 = alloca %struct.point113, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo114([3 x i32] %0) #0 {
  %2 = alloca %struct.point114, align 4
  %3 = alloca %struct.point114, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo115([4 x i32] %0) #0 {
  %2 = alloca %struct.point115, align 4
  %3 = alloca %struct.point115, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo116([4 x i32] %0) #0 {
  %2 = alloca %struct.point116, align 4
  %3 = alloca %struct.point116, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo117([3 x i32] %0) #0 {
  %2 = alloca %struct.point117, align 4
  %3 = alloca %struct.point117, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo118([4 x i32] %0) #0 {
  %2 = alloca %struct.point118, align 4
  %3 = alloca %struct.point118, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo119([4 x i32] %0) #0 {
  %2 = alloca %struct.point119, align 4
  %3 = alloca %struct.point119, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo120([4 x i32] %0) #0 {
  %2 = alloca %struct.point120, align 4
  %3 = alloca %struct.point120, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo121([4 x i32] %0) #0 {
  %2 = alloca %struct.point121, align 4
  %3 = alloca %struct.point121, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo122([3 x i32] %0) #0 {
  %2 = alloca %struct.point122, align 4
  %3 = alloca %struct.point122, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo123([4 x i32] %0) #0 {
  %2 = alloca %struct.point123, align 4
  %3 = alloca %struct.point123, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo124([4 x i32] %0) #0 {
  %2 = alloca %struct.point124, align 4
  %3 = alloca %struct.point124, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo125([4 x i32] %0) #0 {
  %2 = alloca %struct.point125, align 4
  %3 = alloca %struct.point125, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo126([4 x i32] %0) #0 {
  %2 = alloca %struct.point126, align 4
  %3 = alloca %struct.point126, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo127([3 x i32] %0) #0 {
  %2 = alloca %struct.point127, align 4
  %3 = alloca %struct.point127, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo128([3 x i32] %0) #0 {
  %2 = alloca %struct.point128, align 4
  %3 = alloca %struct.point128, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo129([3 x i32] %0) #0 {
  %2 = alloca %struct.point129, align 4
  %3 = alloca %struct.point129, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo130([3 x i32] %0) #0 {
  %2 = alloca %struct.point130, align 4
  %3 = alloca %struct.point130, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo131([4 x i32] %0) #0 {
  %2 = alloca %struct.point131, align 4
  %3 = alloca %struct.point131, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo132([4 x i32] %0) #0 {
  %2 = alloca %struct.point132, align 4
  %3 = alloca %struct.point132, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo133([3 x i32] %0) #0 {
  %2 = alloca %struct.point133, align 4
  %3 = alloca %struct.point133, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo134([4 x i32] %0) #0 {
  %2 = alloca %struct.point134, align 4
  %3 = alloca %struct.point134, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo135([4 x i32] %0) #0 {
  %2 = alloca %struct.point135, align 4
  %3 = alloca %struct.point135, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo136([4 x i32] %0) #0 {
  %2 = alloca %struct.point136, align 4
  %3 = alloca %struct.point136, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo137([4 x i32] %0) #0 {
  %2 = alloca %struct.point137, align 4
  %3 = alloca %struct.point137, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo138([4 x i32] %0) #0 {
  %2 = alloca %struct.point138, align 4
  %3 = alloca %struct.point138, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo139(ptr dead_on_unwind noalias writable sret(%struct.point139) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point139, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo140(ptr dead_on_unwind noalias writable sret(%struct.point140) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point140, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo141(ptr dead_on_unwind noalias writable sret(%struct.point141) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point141, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo142(ptr dead_on_unwind noalias writable sret(%struct.point142) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point142, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo143([4 x i32] %0) #0 {
  %2 = alloca %struct.point143, align 4
  %3 = alloca %struct.point143, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo144(ptr dead_on_unwind noalias writable sret(%struct.point144) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point144, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo145(ptr dead_on_unwind noalias writable sret(%struct.point145) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point145, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo146(ptr dead_on_unwind noalias writable sret(%struct.point146) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point146, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo147(ptr dead_on_unwind noalias writable sret(%struct.point147) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point147, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo148([3 x i32] %0) #0 {
  %2 = alloca %struct.point148, align 4
  %3 = alloca %struct.point148, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo149([3 x i32] %0) #0 {
  %2 = alloca %struct.point149, align 4
  %3 = alloca %struct.point149, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo150([3 x i32] %0) #0 {
  %2 = alloca %struct.point150, align 4
  %3 = alloca %struct.point150, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo151([4 x i32] %0) #0 {
  %2 = alloca %struct.point151, align 4
  %3 = alloca %struct.point151, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo152([4 x i32] %0) #0 {
  %2 = alloca %struct.point152, align 4
  %3 = alloca %struct.point152, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo153([4 x i32] %0) #0 {
  %2 = alloca %struct.point153, align 4
  %3 = alloca %struct.point153, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo154([3 x i32] %0) #0 {
  %2 = alloca %struct.point154, align 4
  %3 = alloca %struct.point154, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo155([4 x i32] %0) #0 {
  %2 = alloca %struct.point155, align 4
  %3 = alloca %struct.point155, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo156([4 x i32] %0) #0 {
  %2 = alloca %struct.point156, align 4
  %3 = alloca %struct.point156, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo157([4 x i32] %0) #0 {
  %2 = alloca %struct.point157, align 4
  %3 = alloca %struct.point157, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo158([4 x i32] %0) #0 {
  %2 = alloca %struct.point158, align 4
  %3 = alloca %struct.point158, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo159([4 x i32] %0) #0 {
  %2 = alloca %struct.point159, align 4
  %3 = alloca %struct.point159, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo160(ptr dead_on_unwind noalias writable sret(%struct.point160) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point160, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo161(ptr dead_on_unwind noalias writable sret(%struct.point161) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point161, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo162(ptr dead_on_unwind noalias writable sret(%struct.point162) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point162, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo163(ptr dead_on_unwind noalias writable sret(%struct.point163) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point163, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo164([4 x i32] %0) #0 {
  %2 = alloca %struct.point164, align 4
  %3 = alloca %struct.point164, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo165(ptr dead_on_unwind noalias writable sret(%struct.point165) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point165, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo166(ptr dead_on_unwind noalias writable sret(%struct.point166) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point166, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo167(ptr dead_on_unwind noalias writable sret(%struct.point167) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point167, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo168(ptr dead_on_unwind noalias writable sret(%struct.point168) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point168, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo169([3 x i32] %0) #0 {
  %2 = alloca %struct.point169, align 4
  %3 = alloca %struct.point169, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo170([4 x i32] %0) #0 {
  %2 = alloca %struct.point170, align 4
  %3 = alloca %struct.point170, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo171([4 x i32] %0) #0 {
  %2 = alloca %struct.point171, align 4
  %3 = alloca %struct.point171, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo172([4 x i32] %0) #0 {
  %2 = alloca %struct.point172, align 4
  %3 = alloca %struct.point172, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo173(ptr dead_on_unwind noalias writable sret(%struct.point173) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point173, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo174(ptr dead_on_unwind noalias writable sret(%struct.point174) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point174, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo175([4 x i32] %0) #0 {
  %2 = alloca %struct.point175, align 4
  %3 = alloca %struct.point175, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo176([4 x i32] %0) #0 {
  %2 = alloca %struct.point176, align 4
  %3 = alloca %struct.point176, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo177([4 x i32] %0) #0 {
  %2 = alloca %struct.point177, align 4
  %3 = alloca %struct.point177, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo178(ptr dead_on_unwind noalias writable sret(%struct.point178) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point178, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo179(ptr dead_on_unwind noalias writable sret(%struct.point179) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point179, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo180([4 x i32] %0) #0 {
  %2 = alloca %struct.point180, align 4
  %3 = alloca %struct.point180, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo181(ptr dead_on_unwind noalias writable sret(%struct.point181) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point181, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo182(ptr dead_on_unwind noalias writable sret(%struct.point182) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point182, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo183(ptr dead_on_unwind noalias writable sret(%struct.point183) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point183, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo184(ptr dead_on_unwind noalias writable sret(%struct.point184) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point184, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo185([4 x i32] %0) #0 {
  %2 = alloca %struct.point185, align 4
  %3 = alloca %struct.point185, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo186(ptr dead_on_unwind noalias writable sret(%struct.point186) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point186, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo187(ptr dead_on_unwind noalias writable sret(%struct.point187) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point187, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo188(ptr dead_on_unwind noalias writable sret(%struct.point188) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point188, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo189(ptr dead_on_unwind noalias writable sret(%struct.point189) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point189, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo190([3 x i32] %0) #0 {
  %2 = alloca %struct.point190, align 4
  %3 = alloca %struct.point190, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo191([4 x i32] %0) #0 {
  %2 = alloca %struct.point191, align 4
  %3 = alloca %struct.point191, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo192([4 x i32] %0) #0 {
  %2 = alloca %struct.point192, align 4
  %3 = alloca %struct.point192, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo193([4 x i32] %0) #0 {
  %2 = alloca %struct.point193, align 4
  %3 = alloca %struct.point193, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo194(ptr dead_on_unwind noalias writable sret(%struct.point194) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point194, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo195(ptr dead_on_unwind noalias writable sret(%struct.point195) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point195, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo196([4 x i32] %0) #0 {
  %2 = alloca %struct.point196, align 4
  %3 = alloca %struct.point196, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo197([4 x i32] %0) #0 {
  %2 = alloca %struct.point197, align 4
  %3 = alloca %struct.point197, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo198([4 x i32] %0) #0 {
  %2 = alloca %struct.point198, align 4
  %3 = alloca %struct.point198, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo199(ptr dead_on_unwind noalias writable sret(%struct.point199) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point199, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo200(ptr dead_on_unwind noalias writable sret(%struct.point200) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point200, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo201([4 x i32] %0) #0 {
  %2 = alloca %struct.point201, align 4
  %3 = alloca %struct.point201, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo202(ptr dead_on_unwind noalias writable sret(%struct.point202) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point202, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo203(ptr dead_on_unwind noalias writable sret(%struct.point203) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point203, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo204(ptr dead_on_unwind noalias writable sret(%struct.point204) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point204, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo205(ptr dead_on_unwind noalias writable sret(%struct.point205) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point205, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo206([4 x i32] %0) #0 {
  %2 = alloca %struct.point206, align 4
  %3 = alloca %struct.point206, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo207(ptr dead_on_unwind noalias writable sret(%struct.point207) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point207, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo208(ptr dead_on_unwind noalias writable sret(%struct.point208) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point208, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo209(ptr dead_on_unwind noalias writable sret(%struct.point209) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point209, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo210(ptr dead_on_unwind noalias writable sret(%struct.point210) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point210, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo211([3 x i32] %0) #0 {
  %2 = alloca %struct.point211, align 4
  %3 = alloca %struct.point211, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo212([3 x i32] %0) #0 {
  %2 = alloca %struct.point212, align 4
  %3 = alloca %struct.point212, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo213([3 x i32] %0) #0 {
  %2 = alloca %struct.point213, align 4
  %3 = alloca %struct.point213, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo214([3 x i32] %0) #0 {
  %2 = alloca %struct.point214, align 4
  %3 = alloca %struct.point214, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo215([4 x i32] %0) #0 {
  %2 = alloca %struct.point215, align 4
  %3 = alloca %struct.point215, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo216([4 x i32] %0) #0 {
  %2 = alloca %struct.point216, align 4
  %3 = alloca %struct.point216, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo217([3 x i32] %0) #0 {
  %2 = alloca %struct.point217, align 4
  %3 = alloca %struct.point217, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo218([4 x i32] %0) #0 {
  %2 = alloca %struct.point218, align 4
  %3 = alloca %struct.point218, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo219([4 x i32] %0) #0 {
  %2 = alloca %struct.point219, align 4
  %3 = alloca %struct.point219, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo220([4 x i32] %0) #0 {
  %2 = alloca %struct.point220, align 4
  %3 = alloca %struct.point220, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo221([4 x i32] %0) #0 {
  %2 = alloca %struct.point221, align 4
  %3 = alloca %struct.point221, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo222([4 x i32] %0) #0 {
  %2 = alloca %struct.point222, align 4
  %3 = alloca %struct.point222, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo223(ptr dead_on_unwind noalias writable sret(%struct.point223) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point223, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo224(ptr dead_on_unwind noalias writable sret(%struct.point224) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point224, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo225(ptr dead_on_unwind noalias writable sret(%struct.point225) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point225, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo226(ptr dead_on_unwind noalias writable sret(%struct.point226) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point226, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo227([4 x i32] %0) #0 {
  %2 = alloca %struct.point227, align 4
  %3 = alloca %struct.point227, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo228(ptr dead_on_unwind noalias writable sret(%struct.point228) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point228, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo229(ptr dead_on_unwind noalias writable sret(%struct.point229) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point229, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo230(ptr dead_on_unwind noalias writable sret(%struct.point230) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point230, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo231(ptr dead_on_unwind noalias writable sret(%struct.point231) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point231, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo232([3 x i32] %0) #0 {
  %2 = alloca %struct.point232, align 4
  %3 = alloca %struct.point232, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo233([3 x i32] %0) #0 {
  %2 = alloca %struct.point233, align 4
  %3 = alloca %struct.point233, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo234([3 x i32] %0) #0 {
  %2 = alloca %struct.point234, align 4
  %3 = alloca %struct.point234, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo235([4 x i32] %0) #0 {
  %2 = alloca %struct.point235, align 4
  %3 = alloca %struct.point235, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo236([4 x i32] %0) #0 {
  %2 = alloca %struct.point236, align 4
  %3 = alloca %struct.point236, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo237([4 x i32] %0) #0 {
  %2 = alloca %struct.point237, align 4
  %3 = alloca %struct.point237, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo238([3 x i32] %0) #0 {
  %2 = alloca %struct.point238, align 4
  %3 = alloca %struct.point238, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo239([4 x i32] %0) #0 {
  %2 = alloca %struct.point239, align 4
  %3 = alloca %struct.point239, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo240([4 x i32] %0) #0 {
  %2 = alloca %struct.point240, align 4
  %3 = alloca %struct.point240, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo241([4 x i32] %0) #0 {
  %2 = alloca %struct.point241, align 4
  %3 = alloca %struct.point241, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo242([4 x i32] %0) #0 {
  %2 = alloca %struct.point242, align 4
  %3 = alloca %struct.point242, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo243([4 x i32] %0) #0 {
  %2 = alloca %struct.point243, align 4
  %3 = alloca %struct.point243, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo244(ptr dead_on_unwind noalias writable sret(%struct.point244) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point244, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo245(ptr dead_on_unwind noalias writable sret(%struct.point245) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point245, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo246(ptr dead_on_unwind noalias writable sret(%struct.point246) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point246, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo247(ptr dead_on_unwind noalias writable sret(%struct.point247) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point247, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo248([4 x i32] %0) #0 {
  %2 = alloca %struct.point248, align 4
  %3 = alloca %struct.point248, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo249(ptr dead_on_unwind noalias writable sret(%struct.point249) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point249, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo250(ptr dead_on_unwind noalias writable sret(%struct.point250) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point250, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo251(ptr dead_on_unwind noalias writable sret(%struct.point251) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point251, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo252(ptr dead_on_unwind noalias writable sret(%struct.point252) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point252, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo253([3 x i32] %0) #0 {
  %2 = alloca %struct.point253, align 4
  %3 = alloca %struct.point253, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo254([4 x i32] %0) #0 {
  %2 = alloca %struct.point254, align 4
  %3 = alloca %struct.point254, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo255([4 x i32] %0) #0 {
  %2 = alloca %struct.point255, align 4
  %3 = alloca %struct.point255, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo256([4 x i32] %0) #0 {
  %2 = alloca %struct.point256, align 4
  %3 = alloca %struct.point256, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo257(ptr dead_on_unwind noalias writable sret(%struct.point257) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point257, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo258(ptr dead_on_unwind noalias writable sret(%struct.point258) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point258, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo259([4 x i32] %0) #0 {
  %2 = alloca %struct.point259, align 4
  %3 = alloca %struct.point259, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo260([4 x i32] %0) #0 {
  %2 = alloca %struct.point260, align 4
  %3 = alloca %struct.point260, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo261([4 x i32] %0) #0 {
  %2 = alloca %struct.point261, align 4
  %3 = alloca %struct.point261, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo262(ptr dead_on_unwind noalias writable sret(%struct.point262) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point262, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo263(ptr dead_on_unwind noalias writable sret(%struct.point263) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point263, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo264([4 x i32] %0) #0 {
  %2 = alloca %struct.point264, align 4
  %3 = alloca %struct.point264, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo265(ptr dead_on_unwind noalias writable sret(%struct.point265) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point265, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo266(ptr dead_on_unwind noalias writable sret(%struct.point266) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point266, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo267(ptr dead_on_unwind noalias writable sret(%struct.point267) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point267, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo268(ptr dead_on_unwind noalias writable sret(%struct.point268) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point268, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo269([4 x i32] %0) #0 {
  %2 = alloca %struct.point269, align 4
  %3 = alloca %struct.point269, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo270(ptr dead_on_unwind noalias writable sret(%struct.point270) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point270, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo271(ptr dead_on_unwind noalias writable sret(%struct.point271) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point271, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo272(ptr dead_on_unwind noalias writable sret(%struct.point272) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point272, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo273(ptr dead_on_unwind noalias writable sret(%struct.point273) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point273, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo274([3 x i32] %0) #0 {
  %2 = alloca %struct.point274, align 4
  %3 = alloca %struct.point274, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo275([4 x i32] %0) #0 {
  %2 = alloca %struct.point275, align 4
  %3 = alloca %struct.point275, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo276([4 x i32] %0) #0 {
  %2 = alloca %struct.point276, align 4
  %3 = alloca %struct.point276, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo277([4 x i32] %0) #0 {
  %2 = alloca %struct.point277, align 4
  %3 = alloca %struct.point277, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo278(ptr dead_on_unwind noalias writable sret(%struct.point278) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point278, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo279(ptr dead_on_unwind noalias writable sret(%struct.point279) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point279, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo280([4 x i32] %0) #0 {
  %2 = alloca %struct.point280, align 4
  %3 = alloca %struct.point280, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo281([4 x i32] %0) #0 {
  %2 = alloca %struct.point281, align 4
  %3 = alloca %struct.point281, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo282([4 x i32] %0) #0 {
  %2 = alloca %struct.point282, align 4
  %3 = alloca %struct.point282, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo283(ptr dead_on_unwind noalias writable sret(%struct.point283) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point283, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo284(ptr dead_on_unwind noalias writable sret(%struct.point284) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point284, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo285([4 x i32] %0) #0 {
  %2 = alloca %struct.point285, align 4
  %3 = alloca %struct.point285, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo286(ptr dead_on_unwind noalias writable sret(%struct.point286) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point286, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo287(ptr dead_on_unwind noalias writable sret(%struct.point287) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point287, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo288(ptr dead_on_unwind noalias writable sret(%struct.point288) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point288, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo289(ptr dead_on_unwind noalias writable sret(%struct.point289) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point289, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo290([4 x i32] %0) #0 {
  %2 = alloca %struct.point290, align 4
  %3 = alloca %struct.point290, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo291(ptr dead_on_unwind noalias writable sret(%struct.point291) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point291, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo292(ptr dead_on_unwind noalias writable sret(%struct.point292) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point292, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo293(ptr dead_on_unwind noalias writable sret(%struct.point293) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point293, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo294(ptr dead_on_unwind noalias writable sret(%struct.point294) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point294, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo295(ptr dead_on_unwind noalias writable sret(%struct.point295) align 2 %0, i32 %1) #0 {
  %3 = alloca %struct.point295, align 2
  store i32 %1, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %0, ptr align 2 %3, i32 4, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo296([2 x i32] %0) #0 {
  %2 = alloca %struct.point296, align 2
  %3 = alloca %struct.point296, align 2
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 6, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo297([2 x i32] %0) #0 {
  %2 = alloca %struct.point297, align 2
  %3 = alloca %struct.point297, align 2
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 6, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo298([2 x i32] %0) #0 {
  %2 = alloca %struct.point298, align 2
  %3 = alloca %struct.point298, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo299([3 x i32] %0) #0 {
  %2 = alloca %struct.point299, align 4
  %3 = alloca %struct.point299, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo300([3 x i32] %0) #0 {
  %2 = alloca %struct.point300, align 4
  %3 = alloca %struct.point300, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo301([2 x i32] %0) #0 {
  %2 = alloca %struct.point301, align 2
  %3 = alloca %struct.point301, align 2
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 6, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo302([2 x i32] %0) #0 {
  %2 = alloca %struct.point302, align 2
  %3 = alloca %struct.point302, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo303([2 x i32] %0) #0 {
  %2 = alloca %struct.point303, align 2
  %3 = alloca %struct.point303, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo304([3 x i32] %0) #0 {
  %2 = alloca %struct.point304, align 4
  %3 = alloca %struct.point304, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo305([3 x i32] %0) #0 {
  %2 = alloca %struct.point305, align 4
  %3 = alloca %struct.point305, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo306([2 x i32] %0) #0 {
  %2 = alloca %struct.point306, align 4
  %3 = alloca %struct.point306, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo307([3 x i32] %0) #0 {
  %2 = alloca %struct.point307, align 4
  %3 = alloca %struct.point307, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo308([3 x i32] %0) #0 {
  %2 = alloca %struct.point308, align 4
  %3 = alloca %struct.point308, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo309([3 x i32] %0) #0 {
  %2 = alloca %struct.point309, align 4
  %3 = alloca %struct.point309, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo310([3 x i32] %0) #0 {
  %2 = alloca %struct.point310, align 4
  %3 = alloca %struct.point310, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo311([2 x i32] %0) #0 {
  %2 = alloca %struct.point311, align 4
  %3 = alloca %struct.point311, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo312([3 x i32] %0) #0 {
  %2 = alloca %struct.point312, align 4
  %3 = alloca %struct.point312, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo313([3 x i32] %0) #0 {
  %2 = alloca %struct.point313, align 4
  %3 = alloca %struct.point313, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo314([3 x i32] %0) #0 {
  %2 = alloca %struct.point314, align 4
  %3 = alloca %struct.point314, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo315([3 x i32] %0) #0 {
  %2 = alloca %struct.point315, align 4
  %3 = alloca %struct.point315, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo316([2 x i32] %0) #0 {
  %2 = alloca %struct.point316, align 2
  %3 = alloca %struct.point316, align 2
  %4 = alloca [2 x i32], align 4
  %5 = alloca [2 x i32], align 4
  store [2 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 6, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 6, i1 false)
  %6 = load [2 x i32], ptr %5, align 4
  ret [2 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo317([2 x i32] %0) #0 {
  %2 = alloca %struct.point317, align 2
  %3 = alloca %struct.point317, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo318([2 x i32] %0) #0 {
  %2 = alloca %struct.point318, align 2
  %3 = alloca %struct.point318, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo319([3 x i32] %0) #0 {
  %2 = alloca %struct.point319, align 2
  %3 = alloca %struct.point319, align 2
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 10, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo320([3 x i32] %0) #0 {
  %2 = alloca %struct.point320, align 4
  %3 = alloca %struct.point320, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo321([3 x i32] %0) #0 {
  %2 = alloca %struct.point321, align 4
  %3 = alloca %struct.point321, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo322([2 x i32] %0) #0 {
  %2 = alloca %struct.point322, align 2
  %3 = alloca %struct.point322, align 2
  store [2 x i32] %0, ptr %3, align 2
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 2
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo323([3 x i32] %0) #0 {
  %2 = alloca %struct.point323, align 2
  %3 = alloca %struct.point323, align 2
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 10, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo324([3 x i32] %0) #0 {
  %2 = alloca %struct.point324, align 2
  %3 = alloca %struct.point324, align 2
  %4 = alloca [3 x i32], align 4
  %5 = alloca [3 x i32], align 4
  store [3 x i32] %0, ptr %4, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %3, ptr align 4 %4, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %2, ptr align 2 %3, i32 10, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %5, ptr align 2 %2, i32 10, i1 false)
  %6 = load [3 x i32], ptr %5, align 4
  ret [3 x i32] %6
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo325([3 x i32] %0) #0 {
  %2 = alloca %struct.point325, align 4
  %3 = alloca %struct.point325, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo326([3 x i32] %0) #0 {
  %2 = alloca %struct.point326, align 4
  %3 = alloca %struct.point326, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo327([3 x i32] %0) #0 {
  %2 = alloca %struct.point327, align 4
  %3 = alloca %struct.point327, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo328([4 x i32] %0) #0 {
  %2 = alloca %struct.point328, align 4
  %3 = alloca %struct.point328, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo329([4 x i32] %0) #0 {
  %2 = alloca %struct.point329, align 4
  %3 = alloca %struct.point329, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo330([4 x i32] %0) #0 {
  %2 = alloca %struct.point330, align 4
  %3 = alloca %struct.point330, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo331([4 x i32] %0) #0 {
  %2 = alloca %struct.point331, align 4
  %3 = alloca %struct.point331, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo332([3 x i32] %0) #0 {
  %2 = alloca %struct.point332, align 4
  %3 = alloca %struct.point332, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo333([4 x i32] %0) #0 {
  %2 = alloca %struct.point333, align 4
  %3 = alloca %struct.point333, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo334([4 x i32] %0) #0 {
  %2 = alloca %struct.point334, align 4
  %3 = alloca %struct.point334, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo335([4 x i32] %0) #0 {
  %2 = alloca %struct.point335, align 4
  %3 = alloca %struct.point335, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo336([4 x i32] %0) #0 {
  %2 = alloca %struct.point336, align 4
  %3 = alloca %struct.point336, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo337([2 x i32] %0) #0 {
  %2 = alloca %struct.point337, align 4
  %3 = alloca %struct.point337, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo338([3 x i32] %0) #0 {
  %2 = alloca %struct.point338, align 4
  %3 = alloca %struct.point338, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo339([3 x i32] %0) #0 {
  %2 = alloca %struct.point339, align 4
  %3 = alloca %struct.point339, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo340([3 x i32] %0) #0 {
  %2 = alloca %struct.point340, align 4
  %3 = alloca %struct.point340, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo341([4 x i32] %0) #0 {
  %2 = alloca %struct.point341, align 4
  %3 = alloca %struct.point341, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo342([4 x i32] %0) #0 {
  %2 = alloca %struct.point342, align 4
  %3 = alloca %struct.point342, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo343([3 x i32] %0) #0 {
  %2 = alloca %struct.point343, align 4
  %3 = alloca %struct.point343, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo344([3 x i32] %0) #0 {
  %2 = alloca %struct.point344, align 4
  %3 = alloca %struct.point344, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo345([3 x i32] %0) #0 {
  %2 = alloca %struct.point345, align 4
  %3 = alloca %struct.point345, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo346([4 x i32] %0) #0 {
  %2 = alloca %struct.point346, align 4
  %3 = alloca %struct.point346, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo347([4 x i32] %0) #0 {
  %2 = alloca %struct.point347, align 4
  %3 = alloca %struct.point347, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo348([3 x i32] %0) #0 {
  %2 = alloca %struct.point348, align 4
  %3 = alloca %struct.point348, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo349([4 x i32] %0) #0 {
  %2 = alloca %struct.point349, align 4
  %3 = alloca %struct.point349, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo350([4 x i32] %0) #0 {
  %2 = alloca %struct.point350, align 4
  %3 = alloca %struct.point350, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo351([4 x i32] %0) #0 {
  %2 = alloca %struct.point351, align 4
  %3 = alloca %struct.point351, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo352([4 x i32] %0) #0 {
  %2 = alloca %struct.point352, align 4
  %3 = alloca %struct.point352, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo353([3 x i32] %0) #0 {
  %2 = alloca %struct.point353, align 4
  %3 = alloca %struct.point353, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo354([4 x i32] %0) #0 {
  %2 = alloca %struct.point354, align 4
  %3 = alloca %struct.point354, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo355([4 x i32] %0) #0 {
  %2 = alloca %struct.point355, align 4
  %3 = alloca %struct.point355, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo356([4 x i32] %0) #0 {
  %2 = alloca %struct.point356, align 4
  %3 = alloca %struct.point356, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo357([4 x i32] %0) #0 {
  %2 = alloca %struct.point357, align 4
  %3 = alloca %struct.point357, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo358([2 x i32] %0) #0 {
  %2 = alloca %struct.point358, align 4
  %3 = alloca %struct.point358, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo359([3 x i32] %0) #0 {
  %2 = alloca %struct.point359, align 4
  %3 = alloca %struct.point359, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo360([3 x i32] %0) #0 {
  %2 = alloca %struct.point360, align 4
  %3 = alloca %struct.point360, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo361([3 x i32] %0) #0 {
  %2 = alloca %struct.point361, align 4
  %3 = alloca %struct.point361, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo362([4 x i32] %0) #0 {
  %2 = alloca %struct.point362, align 4
  %3 = alloca %struct.point362, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo363([4 x i32] %0) #0 {
  %2 = alloca %struct.point363, align 4
  %3 = alloca %struct.point363, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo364([3 x i32] %0) #0 {
  %2 = alloca %struct.point364, align 4
  %3 = alloca %struct.point364, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo365([3 x i32] %0) #0 {
  %2 = alloca %struct.point365, align 4
  %3 = alloca %struct.point365, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo366([3 x i32] %0) #0 {
  %2 = alloca %struct.point366, align 4
  %3 = alloca %struct.point366, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo367([4 x i32] %0) #0 {
  %2 = alloca %struct.point367, align 4
  %3 = alloca %struct.point367, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo368([4 x i32] %0) #0 {
  %2 = alloca %struct.point368, align 4
  %3 = alloca %struct.point368, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo369([3 x i32] %0) #0 {
  %2 = alloca %struct.point369, align 4
  %3 = alloca %struct.point369, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo370([4 x i32] %0) #0 {
  %2 = alloca %struct.point370, align 4
  %3 = alloca %struct.point370, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo371([4 x i32] %0) #0 {
  %2 = alloca %struct.point371, align 4
  %3 = alloca %struct.point371, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo372([4 x i32] %0) #0 {
  %2 = alloca %struct.point372, align 4
  %3 = alloca %struct.point372, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo373([4 x i32] %0) #0 {
  %2 = alloca %struct.point373, align 4
  %3 = alloca %struct.point373, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo374([3 x i32] %0) #0 {
  %2 = alloca %struct.point374, align 4
  %3 = alloca %struct.point374, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo375([4 x i32] %0) #0 {
  %2 = alloca %struct.point375, align 4
  %3 = alloca %struct.point375, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo376([4 x i32] %0) #0 {
  %2 = alloca %struct.point376, align 4
  %3 = alloca %struct.point376, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo377([4 x i32] %0) #0 {
  %2 = alloca %struct.point377, align 4
  %3 = alloca %struct.point377, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo378([4 x i32] %0) #0 {
  %2 = alloca %struct.point378, align 4
  %3 = alloca %struct.point378, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo379([3 x i32] %0) #0 {
  %2 = alloca %struct.point379, align 4
  %3 = alloca %struct.point379, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo380([3 x i32] %0) #0 {
  %2 = alloca %struct.point380, align 4
  %3 = alloca %struct.point380, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo381([3 x i32] %0) #0 {
  %2 = alloca %struct.point381, align 4
  %3 = alloca %struct.point381, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo382([3 x i32] %0) #0 {
  %2 = alloca %struct.point382, align 4
  %3 = alloca %struct.point382, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo383([4 x i32] %0) #0 {
  %2 = alloca %struct.point383, align 4
  %3 = alloca %struct.point383, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo384([4 x i32] %0) #0 {
  %2 = alloca %struct.point384, align 4
  %3 = alloca %struct.point384, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo385([3 x i32] %0) #0 {
  %2 = alloca %struct.point385, align 4
  %3 = alloca %struct.point385, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo386([4 x i32] %0) #0 {
  %2 = alloca %struct.point386, align 4
  %3 = alloca %struct.point386, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo387([4 x i32] %0) #0 {
  %2 = alloca %struct.point387, align 4
  %3 = alloca %struct.point387, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo388([4 x i32] %0) #0 {
  %2 = alloca %struct.point388, align 4
  %3 = alloca %struct.point388, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo389([4 x i32] %0) #0 {
  %2 = alloca %struct.point389, align 4
  %3 = alloca %struct.point389, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo390([4 x i32] %0) #0 {
  %2 = alloca %struct.point390, align 4
  %3 = alloca %struct.point390, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo391(ptr dead_on_unwind noalias writable sret(%struct.point391) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point391, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo392(ptr dead_on_unwind noalias writable sret(%struct.point392) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point392, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo393(ptr dead_on_unwind noalias writable sret(%struct.point393) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point393, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo394(ptr dead_on_unwind noalias writable sret(%struct.point394) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point394, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo395([4 x i32] %0) #0 {
  %2 = alloca %struct.point395, align 4
  %3 = alloca %struct.point395, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo396(ptr dead_on_unwind noalias writable sret(%struct.point396) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point396, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo397(ptr dead_on_unwind noalias writable sret(%struct.point397) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point397, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo398(ptr dead_on_unwind noalias writable sret(%struct.point398) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point398, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo399(ptr dead_on_unwind noalias writable sret(%struct.point399) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point399, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo400([3 x i32] %0) #0 {
  %2 = alloca %struct.point400, align 4
  %3 = alloca %struct.point400, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo401([3 x i32] %0) #0 {
  %2 = alloca %struct.point401, align 4
  %3 = alloca %struct.point401, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo402([3 x i32] %0) #0 {
  %2 = alloca %struct.point402, align 4
  %3 = alloca %struct.point402, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo403([4 x i32] %0) #0 {
  %2 = alloca %struct.point403, align 4
  %3 = alloca %struct.point403, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo404([4 x i32] %0) #0 {
  %2 = alloca %struct.point404, align 4
  %3 = alloca %struct.point404, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo405([4 x i32] %0) #0 {
  %2 = alloca %struct.point405, align 4
  %3 = alloca %struct.point405, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo406([3 x i32] %0) #0 {
  %2 = alloca %struct.point406, align 4
  %3 = alloca %struct.point406, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo407([4 x i32] %0) #0 {
  %2 = alloca %struct.point407, align 4
  %3 = alloca %struct.point407, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo408([4 x i32] %0) #0 {
  %2 = alloca %struct.point408, align 4
  %3 = alloca %struct.point408, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo409([4 x i32] %0) #0 {
  %2 = alloca %struct.point409, align 4
  %3 = alloca %struct.point409, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo410([4 x i32] %0) #0 {
  %2 = alloca %struct.point410, align 4
  %3 = alloca %struct.point410, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo411([4 x i32] %0) #0 {
  %2 = alloca %struct.point411, align 4
  %3 = alloca %struct.point411, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo412(ptr dead_on_unwind noalias writable sret(%struct.point412) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point412, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo413(ptr dead_on_unwind noalias writable sret(%struct.point413) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point413, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo414(ptr dead_on_unwind noalias writable sret(%struct.point414) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point414, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo415(ptr dead_on_unwind noalias writable sret(%struct.point415) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point415, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo416([4 x i32] %0) #0 {
  %2 = alloca %struct.point416, align 4
  %3 = alloca %struct.point416, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo417(ptr dead_on_unwind noalias writable sret(%struct.point417) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point417, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo418(ptr dead_on_unwind noalias writable sret(%struct.point418) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point418, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo419(ptr dead_on_unwind noalias writable sret(%struct.point419) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point419, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo420(ptr dead_on_unwind noalias writable sret(%struct.point420) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point420, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo421([3 x i32] %0) #0 {
  %2 = alloca %struct.point421, align 4
  %3 = alloca %struct.point421, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo422([4 x i32] %0) #0 {
  %2 = alloca %struct.point422, align 4
  %3 = alloca %struct.point422, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo423([4 x i32] %0) #0 {
  %2 = alloca %struct.point423, align 4
  %3 = alloca %struct.point423, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo424([4 x i32] %0) #0 {
  %2 = alloca %struct.point424, align 4
  %3 = alloca %struct.point424, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo425(ptr dead_on_unwind noalias writable sret(%struct.point425) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point425, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo426(ptr dead_on_unwind noalias writable sret(%struct.point426) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point426, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo427([4 x i32] %0) #0 {
  %2 = alloca %struct.point427, align 4
  %3 = alloca %struct.point427, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo428([4 x i32] %0) #0 {
  %2 = alloca %struct.point428, align 4
  %3 = alloca %struct.point428, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo429([4 x i32] %0) #0 {
  %2 = alloca %struct.point429, align 4
  %3 = alloca %struct.point429, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo430(ptr dead_on_unwind noalias writable sret(%struct.point430) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point430, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo431(ptr dead_on_unwind noalias writable sret(%struct.point431) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point431, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo432([4 x i32] %0) #0 {
  %2 = alloca %struct.point432, align 4
  %3 = alloca %struct.point432, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo433(ptr dead_on_unwind noalias writable sret(%struct.point433) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point433, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo434(ptr dead_on_unwind noalias writable sret(%struct.point434) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point434, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo435(ptr dead_on_unwind noalias writable sret(%struct.point435) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point435, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo436(ptr dead_on_unwind noalias writable sret(%struct.point436) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point436, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo437([4 x i32] %0) #0 {
  %2 = alloca %struct.point437, align 4
  %3 = alloca %struct.point437, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo438(ptr dead_on_unwind noalias writable sret(%struct.point438) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point438, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo439(ptr dead_on_unwind noalias writable sret(%struct.point439) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point439, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo440(ptr dead_on_unwind noalias writable sret(%struct.point440) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point440, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo441(ptr dead_on_unwind noalias writable sret(%struct.point441) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point441, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo442([3 x i32] %0) #0 {
  %2 = alloca %struct.point442, align 4
  %3 = alloca %struct.point442, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo443([4 x i32] %0) #0 {
  %2 = alloca %struct.point443, align 4
  %3 = alloca %struct.point443, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo444([4 x i32] %0) #0 {
  %2 = alloca %struct.point444, align 4
  %3 = alloca %struct.point444, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo445([4 x i32] %0) #0 {
  %2 = alloca %struct.point445, align 4
  %3 = alloca %struct.point445, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo446(ptr dead_on_unwind noalias writable sret(%struct.point446) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point446, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo447(ptr dead_on_unwind noalias writable sret(%struct.point447) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point447, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo448([4 x i32] %0) #0 {
  %2 = alloca %struct.point448, align 4
  %3 = alloca %struct.point448, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo449([4 x i32] %0) #0 {
  %2 = alloca %struct.point449, align 4
  %3 = alloca %struct.point449, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo450([4 x i32] %0) #0 {
  %2 = alloca %struct.point450, align 4
  %3 = alloca %struct.point450, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo451(ptr dead_on_unwind noalias writable sret(%struct.point451) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point451, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo452(ptr dead_on_unwind noalias writable sret(%struct.point452) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point452, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo453([4 x i32] %0) #0 {
  %2 = alloca %struct.point453, align 4
  %3 = alloca %struct.point453, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo454(ptr dead_on_unwind noalias writable sret(%struct.point454) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point454, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo455(ptr dead_on_unwind noalias writable sret(%struct.point455) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point455, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo456(ptr dead_on_unwind noalias writable sret(%struct.point456) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point456, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo457(ptr dead_on_unwind noalias writable sret(%struct.point457) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point457, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo458([4 x i32] %0) #0 {
  %2 = alloca %struct.point458, align 4
  %3 = alloca %struct.point458, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo459(ptr dead_on_unwind noalias writable sret(%struct.point459) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point459, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo460(ptr dead_on_unwind noalias writable sret(%struct.point460) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point460, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo461(ptr dead_on_unwind noalias writable sret(%struct.point461) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point461, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo462(ptr dead_on_unwind noalias writable sret(%struct.point462) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point462, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo463([3 x i32] %0) #0 {
  %2 = alloca %struct.point463, align 4
  %3 = alloca %struct.point463, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo464([3 x i32] %0) #0 {
  %2 = alloca %struct.point464, align 4
  %3 = alloca %struct.point464, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo465([3 x i32] %0) #0 {
  %2 = alloca %struct.point465, align 4
  %3 = alloca %struct.point465, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo466([3 x i32] %0) #0 {
  %2 = alloca %struct.point466, align 4
  %3 = alloca %struct.point466, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo467([4 x i32] %0) #0 {
  %2 = alloca %struct.point467, align 4
  %3 = alloca %struct.point467, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo468([4 x i32] %0) #0 {
  %2 = alloca %struct.point468, align 4
  %3 = alloca %struct.point468, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo469([3 x i32] %0) #0 {
  %2 = alloca %struct.point469, align 4
  %3 = alloca %struct.point469, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo470([4 x i32] %0) #0 {
  %2 = alloca %struct.point470, align 4
  %3 = alloca %struct.point470, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo471([4 x i32] %0) #0 {
  %2 = alloca %struct.point471, align 4
  %3 = alloca %struct.point471, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo472([4 x i32] %0) #0 {
  %2 = alloca %struct.point472, align 4
  %3 = alloca %struct.point472, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo473([4 x i32] %0) #0 {
  %2 = alloca %struct.point473, align 4
  %3 = alloca %struct.point473, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo474([4 x i32] %0) #0 {
  %2 = alloca %struct.point474, align 4
  %3 = alloca %struct.point474, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo475(ptr dead_on_unwind noalias writable sret(%struct.point475) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point475, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo476(ptr dead_on_unwind noalias writable sret(%struct.point476) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point476, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo477(ptr dead_on_unwind noalias writable sret(%struct.point477) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point477, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo478(ptr dead_on_unwind noalias writable sret(%struct.point478) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point478, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo479([4 x i32] %0) #0 {
  %2 = alloca %struct.point479, align 4
  %3 = alloca %struct.point479, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo480(ptr dead_on_unwind noalias writable sret(%struct.point480) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point480, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo481(ptr dead_on_unwind noalias writable sret(%struct.point481) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point481, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo482(ptr dead_on_unwind noalias writable sret(%struct.point482) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point482, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo483(ptr dead_on_unwind noalias writable sret(%struct.point483) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point483, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo484([3 x i32] %0) #0 {
  %2 = alloca %struct.point484, align 4
  %3 = alloca %struct.point484, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo485([3 x i32] %0) #0 {
  %2 = alloca %struct.point485, align 4
  %3 = alloca %struct.point485, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo486([3 x i32] %0) #0 {
  %2 = alloca %struct.point486, align 4
  %3 = alloca %struct.point486, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo487([4 x i32] %0) #0 {
  %2 = alloca %struct.point487, align 4
  %3 = alloca %struct.point487, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo488([4 x i32] %0) #0 {
  %2 = alloca %struct.point488, align 4
  %3 = alloca %struct.point488, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo489([4 x i32] %0) #0 {
  %2 = alloca %struct.point489, align 4
  %3 = alloca %struct.point489, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo490([3 x i32] %0) #0 {
  %2 = alloca %struct.point490, align 4
  %3 = alloca %struct.point490, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo491([4 x i32] %0) #0 {
  %2 = alloca %struct.point491, align 4
  %3 = alloca %struct.point491, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo492([4 x i32] %0) #0 {
  %2 = alloca %struct.point492, align 4
  %3 = alloca %struct.point492, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo493([4 x i32] %0) #0 {
  %2 = alloca %struct.point493, align 4
  %3 = alloca %struct.point493, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo494([4 x i32] %0) #0 {
  %2 = alloca %struct.point494, align 4
  %3 = alloca %struct.point494, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo495([4 x i32] %0) #0 {
  %2 = alloca %struct.point495, align 4
  %3 = alloca %struct.point495, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo496(ptr dead_on_unwind noalias writable sret(%struct.point496) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point496, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo497(ptr dead_on_unwind noalias writable sret(%struct.point497) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point497, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo498(ptr dead_on_unwind noalias writable sret(%struct.point498) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point498, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo499(ptr dead_on_unwind noalias writable sret(%struct.point499) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point499, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo500([4 x i32] %0) #0 {
  %2 = alloca %struct.point500, align 4
  %3 = alloca %struct.point500, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo501(ptr dead_on_unwind noalias writable sret(%struct.point501) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point501, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo502(ptr dead_on_unwind noalias writable sret(%struct.point502) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point502, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo503(ptr dead_on_unwind noalias writable sret(%struct.point503) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point503, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo504(ptr dead_on_unwind noalias writable sret(%struct.point504) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point504, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo505([3 x i32] %0) #0 {
  %2 = alloca %struct.point505, align 4
  %3 = alloca %struct.point505, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo506([4 x i32] %0) #0 {
  %2 = alloca %struct.point506, align 4
  %3 = alloca %struct.point506, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo507([4 x i32] %0) #0 {
  %2 = alloca %struct.point507, align 4
  %3 = alloca %struct.point507, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo508([4 x i32] %0) #0 {
  %2 = alloca %struct.point508, align 4
  %3 = alloca %struct.point508, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo509(ptr dead_on_unwind noalias writable sret(%struct.point509) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point509, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo510(ptr dead_on_unwind noalias writable sret(%struct.point510) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point510, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo511([4 x i32] %0) #0 {
  %2 = alloca %struct.point511, align 4
  %3 = alloca %struct.point511, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo512([4 x i32] %0) #0 {
  %2 = alloca %struct.point512, align 4
  %3 = alloca %struct.point512, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo513([4 x i32] %0) #0 {
  %2 = alloca %struct.point513, align 4
  %3 = alloca %struct.point513, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo514(ptr dead_on_unwind noalias writable sret(%struct.point514) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point514, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo515(ptr dead_on_unwind noalias writable sret(%struct.point515) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point515, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo516([4 x i32] %0) #0 {
  %2 = alloca %struct.point516, align 4
  %3 = alloca %struct.point516, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo517(ptr dead_on_unwind noalias writable sret(%struct.point517) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point517, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo518(ptr dead_on_unwind noalias writable sret(%struct.point518) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point518, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo519(ptr dead_on_unwind noalias writable sret(%struct.point519) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point519, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo520(ptr dead_on_unwind noalias writable sret(%struct.point520) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point520, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo521([4 x i32] %0) #0 {
  %2 = alloca %struct.point521, align 4
  %3 = alloca %struct.point521, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo522(ptr dead_on_unwind noalias writable sret(%struct.point522) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point522, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo523(ptr dead_on_unwind noalias writable sret(%struct.point523) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point523, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo524(ptr dead_on_unwind noalias writable sret(%struct.point524) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point524, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo525(ptr dead_on_unwind noalias writable sret(%struct.point525) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point525, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo526([3 x i32] %0) #0 {
  %2 = alloca %struct.point526, align 4
  %3 = alloca %struct.point526, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo527([4 x i32] %0) #0 {
  %2 = alloca %struct.point527, align 4
  %3 = alloca %struct.point527, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo528([4 x i32] %0) #0 {
  %2 = alloca %struct.point528, align 4
  %3 = alloca %struct.point528, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo529([4 x i32] %0) #0 {
  %2 = alloca %struct.point529, align 4
  %3 = alloca %struct.point529, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo530(ptr dead_on_unwind noalias writable sret(%struct.point530) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point530, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo531(ptr dead_on_unwind noalias writable sret(%struct.point531) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point531, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo532([4 x i32] %0) #0 {
  %2 = alloca %struct.point532, align 4
  %3 = alloca %struct.point532, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo533([4 x i32] %0) #0 {
  %2 = alloca %struct.point533, align 4
  %3 = alloca %struct.point533, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo534([4 x i32] %0) #0 {
  %2 = alloca %struct.point534, align 4
  %3 = alloca %struct.point534, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo535(ptr dead_on_unwind noalias writable sret(%struct.point535) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point535, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo536(ptr dead_on_unwind noalias writable sret(%struct.point536) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point536, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo537([4 x i32] %0) #0 {
  %2 = alloca %struct.point537, align 4
  %3 = alloca %struct.point537, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo538(ptr dead_on_unwind noalias writable sret(%struct.point538) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point538, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo539(ptr dead_on_unwind noalias writable sret(%struct.point539) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point539, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo540(ptr dead_on_unwind noalias writable sret(%struct.point540) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point540, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo541(ptr dead_on_unwind noalias writable sret(%struct.point541) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point541, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo542([4 x i32] %0) #0 {
  %2 = alloca %struct.point542, align 4
  %3 = alloca %struct.point542, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo543(ptr dead_on_unwind noalias writable sret(%struct.point543) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point543, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo544(ptr dead_on_unwind noalias writable sret(%struct.point544) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point544, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo545(ptr dead_on_unwind noalias writable sret(%struct.point545) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point545, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo546(ptr dead_on_unwind noalias writable sret(%struct.point546) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point546, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo547([2 x i32] %0) #0 {
  %2 = alloca %struct.point547, align 4
  %3 = alloca %struct.point547, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo548([2 x i32] %0) #0 {
  %2 = alloca %struct.point548, align 4
  %3 = alloca %struct.point548, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo549([2 x i32] %0) #0 {
  %2 = alloca %struct.point549, align 4
  %3 = alloca %struct.point549, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo550([3 x i32] %0) #0 {
  %2 = alloca %struct.point550, align 4
  %3 = alloca %struct.point550, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo551([3 x i32] %0) #0 {
  %2 = alloca %struct.point551, align 4
  %3 = alloca %struct.point551, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo552([3 x i32] %0) #0 {
  %2 = alloca %struct.point552, align 4
  %3 = alloca %struct.point552, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo553([2 x i32] %0) #0 {
  %2 = alloca %struct.point553, align 4
  %3 = alloca %struct.point553, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo554([3 x i32] %0) #0 {
  %2 = alloca %struct.point554, align 4
  %3 = alloca %struct.point554, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo555([3 x i32] %0) #0 {
  %2 = alloca %struct.point555, align 4
  %3 = alloca %struct.point555, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo556([3 x i32] %0) #0 {
  %2 = alloca %struct.point556, align 4
  %3 = alloca %struct.point556, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo557([3 x i32] %0) #0 {
  %2 = alloca %struct.point557, align 4
  %3 = alloca %struct.point557, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo558([3 x i32] %0) #0 {
  %2 = alloca %struct.point558, align 4
  %3 = alloca %struct.point558, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo559([4 x i32] %0) #0 {
  %2 = alloca %struct.point559, align 4
  %3 = alloca %struct.point559, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo560([4 x i32] %0) #0 {
  %2 = alloca %struct.point560, align 4
  %3 = alloca %struct.point560, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo561([4 x i32] %0) #0 {
  %2 = alloca %struct.point561, align 4
  %3 = alloca %struct.point561, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo562([4 x i32] %0) #0 {
  %2 = alloca %struct.point562, align 4
  %3 = alloca %struct.point562, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo563([3 x i32] %0) #0 {
  %2 = alloca %struct.point563, align 4
  %3 = alloca %struct.point563, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo564([4 x i32] %0) #0 {
  %2 = alloca %struct.point564, align 4
  %3 = alloca %struct.point564, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo565([4 x i32] %0) #0 {
  %2 = alloca %struct.point565, align 4
  %3 = alloca %struct.point565, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo566([4 x i32] %0) #0 {
  %2 = alloca %struct.point566, align 4
  %3 = alloca %struct.point566, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo567([4 x i32] %0) #0 {
  %2 = alloca %struct.point567, align 4
  %3 = alloca %struct.point567, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo568([2 x i32] %0) #0 {
  %2 = alloca %struct.point568, align 4
  %3 = alloca %struct.point568, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo569([3 x i32] %0) #0 {
  %2 = alloca %struct.point569, align 4
  %3 = alloca %struct.point569, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo570([3 x i32] %0) #0 {
  %2 = alloca %struct.point570, align 4
  %3 = alloca %struct.point570, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo571([3 x i32] %0) #0 {
  %2 = alloca %struct.point571, align 4
  %3 = alloca %struct.point571, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo572([4 x i32] %0) #0 {
  %2 = alloca %struct.point572, align 4
  %3 = alloca %struct.point572, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo573([4 x i32] %0) #0 {
  %2 = alloca %struct.point573, align 4
  %3 = alloca %struct.point573, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo574([3 x i32] %0) #0 {
  %2 = alloca %struct.point574, align 4
  %3 = alloca %struct.point574, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo575([3 x i32] %0) #0 {
  %2 = alloca %struct.point575, align 4
  %3 = alloca %struct.point575, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo576([3 x i32] %0) #0 {
  %2 = alloca %struct.point576, align 4
  %3 = alloca %struct.point576, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo577([4 x i32] %0) #0 {
  %2 = alloca %struct.point577, align 4
  %3 = alloca %struct.point577, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo578([4 x i32] %0) #0 {
  %2 = alloca %struct.point578, align 4
  %3 = alloca %struct.point578, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo579([3 x i32] %0) #0 {
  %2 = alloca %struct.point579, align 4
  %3 = alloca %struct.point579, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo580([4 x i32] %0) #0 {
  %2 = alloca %struct.point580, align 4
  %3 = alloca %struct.point580, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo581([4 x i32] %0) #0 {
  %2 = alloca %struct.point581, align 4
  %3 = alloca %struct.point581, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo582([4 x i32] %0) #0 {
  %2 = alloca %struct.point582, align 4
  %3 = alloca %struct.point582, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo583([4 x i32] %0) #0 {
  %2 = alloca %struct.point583, align 4
  %3 = alloca %struct.point583, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo584([3 x i32] %0) #0 {
  %2 = alloca %struct.point584, align 4
  %3 = alloca %struct.point584, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo585([4 x i32] %0) #0 {
  %2 = alloca %struct.point585, align 4
  %3 = alloca %struct.point585, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo586([4 x i32] %0) #0 {
  %2 = alloca %struct.point586, align 4
  %3 = alloca %struct.point586, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo587([4 x i32] %0) #0 {
  %2 = alloca %struct.point587, align 4
  %3 = alloca %struct.point587, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo588([4 x i32] %0) #0 {
  %2 = alloca %struct.point588, align 4
  %3 = alloca %struct.point588, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo589([3 x i32] %0) #0 {
  %2 = alloca %struct.point589, align 4
  %3 = alloca %struct.point589, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo590([4 x i32] %0) #0 {
  %2 = alloca %struct.point590, align 4
  %3 = alloca %struct.point590, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo591([4 x i32] %0) #0 {
  %2 = alloca %struct.point591, align 4
  %3 = alloca %struct.point591, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo592([4 x i32] %0) #0 {
  %2 = alloca %struct.point592, align 4
  %3 = alloca %struct.point592, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo593(ptr dead_on_unwind noalias writable sret(%struct.point593) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point593, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo594(ptr dead_on_unwind noalias writable sret(%struct.point594) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point594, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo595([4 x i32] %0) #0 {
  %2 = alloca %struct.point595, align 4
  %3 = alloca %struct.point595, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo596([4 x i32] %0) #0 {
  %2 = alloca %struct.point596, align 4
  %3 = alloca %struct.point596, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo597([4 x i32] %0) #0 {
  %2 = alloca %struct.point597, align 4
  %3 = alloca %struct.point597, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo598(ptr dead_on_unwind noalias writable sret(%struct.point598) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point598, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo599(ptr dead_on_unwind noalias writable sret(%struct.point599) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point599, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo600([4 x i32] %0) #0 {
  %2 = alloca %struct.point600, align 4
  %3 = alloca %struct.point600, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo601(ptr dead_on_unwind noalias writable sret(%struct.point601) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point601, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo602(ptr dead_on_unwind noalias writable sret(%struct.point602) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point602, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo603(ptr dead_on_unwind noalias writable sret(%struct.point603) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point603, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo604(ptr dead_on_unwind noalias writable sret(%struct.point604) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point604, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo605([4 x i32] %0) #0 {
  %2 = alloca %struct.point605, align 4
  %3 = alloca %struct.point605, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo606(ptr dead_on_unwind noalias writable sret(%struct.point606) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point606, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo607(ptr dead_on_unwind noalias writable sret(%struct.point607) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point607, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo608(ptr dead_on_unwind noalias writable sret(%struct.point608) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point608, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo609(ptr dead_on_unwind noalias writable sret(%struct.point609) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point609, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo610([3 x i32] %0) #0 {
  %2 = alloca %struct.point610, align 4
  %3 = alloca %struct.point610, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo611([4 x i32] %0) #0 {
  %2 = alloca %struct.point611, align 4
  %3 = alloca %struct.point611, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo612([4 x i32] %0) #0 {
  %2 = alloca %struct.point612, align 4
  %3 = alloca %struct.point612, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo613([4 x i32] %0) #0 {
  %2 = alloca %struct.point613, align 4
  %3 = alloca %struct.point613, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo614(ptr dead_on_unwind noalias writable sret(%struct.point614) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point614, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo615(ptr dead_on_unwind noalias writable sret(%struct.point615) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point615, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo616([4 x i32] %0) #0 {
  %2 = alloca %struct.point616, align 4
  %3 = alloca %struct.point616, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo617([4 x i32] %0) #0 {
  %2 = alloca %struct.point617, align 4
  %3 = alloca %struct.point617, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo618([4 x i32] %0) #0 {
  %2 = alloca %struct.point618, align 4
  %3 = alloca %struct.point618, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo619(ptr dead_on_unwind noalias writable sret(%struct.point619) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point619, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo620(ptr dead_on_unwind noalias writable sret(%struct.point620) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point620, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo621([4 x i32] %0) #0 {
  %2 = alloca %struct.point621, align 4
  %3 = alloca %struct.point621, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo622(ptr dead_on_unwind noalias writable sret(%struct.point622) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point622, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo623(ptr dead_on_unwind noalias writable sret(%struct.point623) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point623, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo624(ptr dead_on_unwind noalias writable sret(%struct.point624) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point624, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo625(ptr dead_on_unwind noalias writable sret(%struct.point625) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point625, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo626([4 x i32] %0) #0 {
  %2 = alloca %struct.point626, align 4
  %3 = alloca %struct.point626, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo627(ptr dead_on_unwind noalias writable sret(%struct.point627) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point627, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo628(ptr dead_on_unwind noalias writable sret(%struct.point628) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point628, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo629(ptr dead_on_unwind noalias writable sret(%struct.point629) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point629, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo630(ptr dead_on_unwind noalias writable sret(%struct.point630) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point630, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo631([2 x i32] %0) #0 {
  %2 = alloca %struct.point631, align 4
  %3 = alloca %struct.point631, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo632([2 x i32] %0) #0 {
  %2 = alloca %struct.point632, align 4
  %3 = alloca %struct.point632, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo633([3 x i32] %0) #0 {
  %2 = alloca %struct.point633, align 4
  %3 = alloca %struct.point633, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo634([3 x i32] %0) #0 {
  %2 = alloca %struct.point634, align 4
  %3 = alloca %struct.point634, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo635([3 x i32] %0) #0 {
  %2 = alloca %struct.point635, align 4
  %3 = alloca %struct.point635, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo636([3 x i32] %0) #0 {
  %2 = alloca %struct.point636, align 4
  %3 = alloca %struct.point636, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo637([3 x i32] %0) #0 {
  %2 = alloca %struct.point637, align 4
  %3 = alloca %struct.point637, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo638([3 x i32] %0) #0 {
  %2 = alloca %struct.point638, align 4
  %3 = alloca %struct.point638, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo639([3 x i32] %0) #0 {
  %2 = alloca %struct.point639, align 4
  %3 = alloca %struct.point639, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo640([4 x i32] %0) #0 {
  %2 = alloca %struct.point640, align 4
  %3 = alloca %struct.point640, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo641([4 x i32] %0) #0 {
  %2 = alloca %struct.point641, align 4
  %3 = alloca %struct.point641, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo642([3 x i32] %0) #0 {
  %2 = alloca %struct.point642, align 4
  %3 = alloca %struct.point642, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo643([4 x i32] %0) #0 {
  %2 = alloca %struct.point643, align 4
  %3 = alloca %struct.point643, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo644([4 x i32] %0) #0 {
  %2 = alloca %struct.point644, align 4
  %3 = alloca %struct.point644, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo645([4 x i32] %0) #0 {
  %2 = alloca %struct.point645, align 4
  %3 = alloca %struct.point645, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo646([4 x i32] %0) #0 {
  %2 = alloca %struct.point646, align 4
  %3 = alloca %struct.point646, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo647([3 x i32] %0) #0 {
  %2 = alloca %struct.point647, align 4
  %3 = alloca %struct.point647, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo648([4 x i32] %0) #0 {
  %2 = alloca %struct.point648, align 4
  %3 = alloca %struct.point648, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo649([4 x i32] %0) #0 {
  %2 = alloca %struct.point649, align 4
  %3 = alloca %struct.point649, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo650([4 x i32] %0) #0 {
  %2 = alloca %struct.point650, align 4
  %3 = alloca %struct.point650, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo651([4 x i32] %0) #0 {
  %2 = alloca %struct.point651, align 4
  %3 = alloca %struct.point651, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo652([2 x i32] %0) #0 {
  %2 = alloca %struct.point652, align 4
  %3 = alloca %struct.point652, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo653([3 x i32] %0) #0 {
  %2 = alloca %struct.point653, align 4
  %3 = alloca %struct.point653, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo654([3 x i32] %0) #0 {
  %2 = alloca %struct.point654, align 4
  %3 = alloca %struct.point654, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo655([3 x i32] %0) #0 {
  %2 = alloca %struct.point655, align 4
  %3 = alloca %struct.point655, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo656([4 x i32] %0) #0 {
  %2 = alloca %struct.point656, align 4
  %3 = alloca %struct.point656, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo657([4 x i32] %0) #0 {
  %2 = alloca %struct.point657, align 4
  %3 = alloca %struct.point657, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo658([3 x i32] %0) #0 {
  %2 = alloca %struct.point658, align 4
  %3 = alloca %struct.point658, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo659([3 x i32] %0) #0 {
  %2 = alloca %struct.point659, align 4
  %3 = alloca %struct.point659, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo660([3 x i32] %0) #0 {
  %2 = alloca %struct.point660, align 4
  %3 = alloca %struct.point660, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo661([4 x i32] %0) #0 {
  %2 = alloca %struct.point661, align 4
  %3 = alloca %struct.point661, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo662([4 x i32] %0) #0 {
  %2 = alloca %struct.point662, align 4
  %3 = alloca %struct.point662, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo663([3 x i32] %0) #0 {
  %2 = alloca %struct.point663, align 4
  %3 = alloca %struct.point663, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo664([4 x i32] %0) #0 {
  %2 = alloca %struct.point664, align 4
  %3 = alloca %struct.point664, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo665([4 x i32] %0) #0 {
  %2 = alloca %struct.point665, align 4
  %3 = alloca %struct.point665, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo666([4 x i32] %0) #0 {
  %2 = alloca %struct.point666, align 4
  %3 = alloca %struct.point666, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo667([4 x i32] %0) #0 {
  %2 = alloca %struct.point667, align 4
  %3 = alloca %struct.point667, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo668([3 x i32] %0) #0 {
  %2 = alloca %struct.point668, align 4
  %3 = alloca %struct.point668, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo669([4 x i32] %0) #0 {
  %2 = alloca %struct.point669, align 4
  %3 = alloca %struct.point669, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo670([4 x i32] %0) #0 {
  %2 = alloca %struct.point670, align 4
  %3 = alloca %struct.point670, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo671([4 x i32] %0) #0 {
  %2 = alloca %struct.point671, align 4
  %3 = alloca %struct.point671, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo672([4 x i32] %0) #0 {
  %2 = alloca %struct.point672, align 4
  %3 = alloca %struct.point672, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo673([3 x i32] %0) #0 {
  %2 = alloca %struct.point673, align 4
  %3 = alloca %struct.point673, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo674([4 x i32] %0) #0 {
  %2 = alloca %struct.point674, align 4
  %3 = alloca %struct.point674, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo675([4 x i32] %0) #0 {
  %2 = alloca %struct.point675, align 4
  %3 = alloca %struct.point675, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo676([4 x i32] %0) #0 {
  %2 = alloca %struct.point676, align 4
  %3 = alloca %struct.point676, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo677(ptr dead_on_unwind noalias writable sret(%struct.point677) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point677, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo678(ptr dead_on_unwind noalias writable sret(%struct.point678) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point678, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo679([4 x i32] %0) #0 {
  %2 = alloca %struct.point679, align 4
  %3 = alloca %struct.point679, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo680([4 x i32] %0) #0 {
  %2 = alloca %struct.point680, align 4
  %3 = alloca %struct.point680, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo681([4 x i32] %0) #0 {
  %2 = alloca %struct.point681, align 4
  %3 = alloca %struct.point681, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo682(ptr dead_on_unwind noalias writable sret(%struct.point682) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point682, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo683(ptr dead_on_unwind noalias writable sret(%struct.point683) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point683, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo684([4 x i32] %0) #0 {
  %2 = alloca %struct.point684, align 4
  %3 = alloca %struct.point684, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo685(ptr dead_on_unwind noalias writable sret(%struct.point685) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point685, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo686(ptr dead_on_unwind noalias writable sret(%struct.point686) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point686, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo687(ptr dead_on_unwind noalias writable sret(%struct.point687) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point687, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo688(ptr dead_on_unwind noalias writable sret(%struct.point688) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point688, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo689([4 x i32] %0) #0 {
  %2 = alloca %struct.point689, align 4
  %3 = alloca %struct.point689, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo690(ptr dead_on_unwind noalias writable sret(%struct.point690) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point690, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo691(ptr dead_on_unwind noalias writable sret(%struct.point691) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point691, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo692(ptr dead_on_unwind noalias writable sret(%struct.point692) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point692, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo693(ptr dead_on_unwind noalias writable sret(%struct.point693) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point693, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo694([3 x i32] %0) #0 {
  %2 = alloca %struct.point694, align 4
  %3 = alloca %struct.point694, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo695([4 x i32] %0) #0 {
  %2 = alloca %struct.point695, align 4
  %3 = alloca %struct.point695, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo696([4 x i32] %0) #0 {
  %2 = alloca %struct.point696, align 4
  %3 = alloca %struct.point696, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo697([4 x i32] %0) #0 {
  %2 = alloca %struct.point697, align 4
  %3 = alloca %struct.point697, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo698(ptr dead_on_unwind noalias writable sret(%struct.point698) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point698, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo699(ptr dead_on_unwind noalias writable sret(%struct.point699) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point699, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo700([4 x i32] %0) #0 {
  %2 = alloca %struct.point700, align 4
  %3 = alloca %struct.point700, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo701([4 x i32] %0) #0 {
  %2 = alloca %struct.point701, align 4
  %3 = alloca %struct.point701, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo702([4 x i32] %0) #0 {
  %2 = alloca %struct.point702, align 4
  %3 = alloca %struct.point702, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo703(ptr dead_on_unwind noalias writable sret(%struct.point703) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point703, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo704(ptr dead_on_unwind noalias writable sret(%struct.point704) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point704, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo705([4 x i32] %0) #0 {
  %2 = alloca %struct.point705, align 4
  %3 = alloca %struct.point705, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo706(ptr dead_on_unwind noalias writable sret(%struct.point706) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point706, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo707(ptr dead_on_unwind noalias writable sret(%struct.point707) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point707, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo708(ptr dead_on_unwind noalias writable sret(%struct.point708) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point708, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo709(ptr dead_on_unwind noalias writable sret(%struct.point709) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point709, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo710([4 x i32] %0) #0 {
  %2 = alloca %struct.point710, align 4
  %3 = alloca %struct.point710, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo711(ptr dead_on_unwind noalias writable sret(%struct.point711) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point711, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo712(ptr dead_on_unwind noalias writable sret(%struct.point712) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point712, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo713(ptr dead_on_unwind noalias writable sret(%struct.point713) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point713, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo714(ptr dead_on_unwind noalias writable sret(%struct.point714) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point714, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo715([3 x i32] %0) #0 {
  %2 = alloca %struct.point715, align 4
  %3 = alloca %struct.point715, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo716([3 x i32] %0) #0 {
  %2 = alloca %struct.point716, align 4
  %3 = alloca %struct.point716, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo717([3 x i32] %0) #0 {
  %2 = alloca %struct.point717, align 4
  %3 = alloca %struct.point717, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo718([3 x i32] %0) #0 {
  %2 = alloca %struct.point718, align 4
  %3 = alloca %struct.point718, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo719([4 x i32] %0) #0 {
  %2 = alloca %struct.point719, align 4
  %3 = alloca %struct.point719, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo720([4 x i32] %0) #0 {
  %2 = alloca %struct.point720, align 4
  %3 = alloca %struct.point720, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo721([3 x i32] %0) #0 {
  %2 = alloca %struct.point721, align 4
  %3 = alloca %struct.point721, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo722([4 x i32] %0) #0 {
  %2 = alloca %struct.point722, align 4
  %3 = alloca %struct.point722, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo723([4 x i32] %0) #0 {
  %2 = alloca %struct.point723, align 4
  %3 = alloca %struct.point723, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo724([4 x i32] %0) #0 {
  %2 = alloca %struct.point724, align 4
  %3 = alloca %struct.point724, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo725([4 x i32] %0) #0 {
  %2 = alloca %struct.point725, align 4
  %3 = alloca %struct.point725, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo726([4 x i32] %0) #0 {
  %2 = alloca %struct.point726, align 4
  %3 = alloca %struct.point726, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo727(ptr dead_on_unwind noalias writable sret(%struct.point727) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point727, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo728(ptr dead_on_unwind noalias writable sret(%struct.point728) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point728, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo729(ptr dead_on_unwind noalias writable sret(%struct.point729) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point729, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo730(ptr dead_on_unwind noalias writable sret(%struct.point730) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point730, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo731([4 x i32] %0) #0 {
  %2 = alloca %struct.point731, align 4
  %3 = alloca %struct.point731, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo732(ptr dead_on_unwind noalias writable sret(%struct.point732) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point732, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo733(ptr dead_on_unwind noalias writable sret(%struct.point733) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point733, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo734(ptr dead_on_unwind noalias writable sret(%struct.point734) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point734, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo735(ptr dead_on_unwind noalias writable sret(%struct.point735) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point735, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo736([3 x i32] %0) #0 {
  %2 = alloca %struct.point736, align 4
  %3 = alloca %struct.point736, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo737([3 x i32] %0) #0 {
  %2 = alloca %struct.point737, align 4
  %3 = alloca %struct.point737, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo738([3 x i32] %0) #0 {
  %2 = alloca %struct.point738, align 4
  %3 = alloca %struct.point738, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo739([4 x i32] %0) #0 {
  %2 = alloca %struct.point739, align 4
  %3 = alloca %struct.point739, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo740([4 x i32] %0) #0 {
  %2 = alloca %struct.point740, align 4
  %3 = alloca %struct.point740, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo741([4 x i32] %0) #0 {
  %2 = alloca %struct.point741, align 4
  %3 = alloca %struct.point741, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo742([3 x i32] %0) #0 {
  %2 = alloca %struct.point742, align 4
  %3 = alloca %struct.point742, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo743([4 x i32] %0) #0 {
  %2 = alloca %struct.point743, align 4
  %3 = alloca %struct.point743, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo744([4 x i32] %0) #0 {
  %2 = alloca %struct.point744, align 4
  %3 = alloca %struct.point744, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo745([4 x i32] %0) #0 {
  %2 = alloca %struct.point745, align 4
  %3 = alloca %struct.point745, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo746([4 x i32] %0) #0 {
  %2 = alloca %struct.point746, align 4
  %3 = alloca %struct.point746, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo747([4 x i32] %0) #0 {
  %2 = alloca %struct.point747, align 4
  %3 = alloca %struct.point747, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo748(ptr dead_on_unwind noalias writable sret(%struct.point748) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point748, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo749(ptr dead_on_unwind noalias writable sret(%struct.point749) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point749, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo750(ptr dead_on_unwind noalias writable sret(%struct.point750) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point750, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo751(ptr dead_on_unwind noalias writable sret(%struct.point751) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point751, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo752([4 x i32] %0) #0 {
  %2 = alloca %struct.point752, align 4
  %3 = alloca %struct.point752, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo753(ptr dead_on_unwind noalias writable sret(%struct.point753) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point753, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo754(ptr dead_on_unwind noalias writable sret(%struct.point754) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point754, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo755(ptr dead_on_unwind noalias writable sret(%struct.point755) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point755, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo756(ptr dead_on_unwind noalias writable sret(%struct.point756) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point756, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo757([3 x i32] %0) #0 {
  %2 = alloca %struct.point757, align 4
  %3 = alloca %struct.point757, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo758([4 x i32] %0) #0 {
  %2 = alloca %struct.point758, align 4
  %3 = alloca %struct.point758, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo759([4 x i32] %0) #0 {
  %2 = alloca %struct.point759, align 4
  %3 = alloca %struct.point759, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo760([4 x i32] %0) #0 {
  %2 = alloca %struct.point760, align 4
  %3 = alloca %struct.point760, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo761(ptr dead_on_unwind noalias writable sret(%struct.point761) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point761, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo762(ptr dead_on_unwind noalias writable sret(%struct.point762) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point762, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo763([4 x i32] %0) #0 {
  %2 = alloca %struct.point763, align 4
  %3 = alloca %struct.point763, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo764([4 x i32] %0) #0 {
  %2 = alloca %struct.point764, align 4
  %3 = alloca %struct.point764, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo765([4 x i32] %0) #0 {
  %2 = alloca %struct.point765, align 4
  %3 = alloca %struct.point765, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo766(ptr dead_on_unwind noalias writable sret(%struct.point766) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point766, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo767(ptr dead_on_unwind noalias writable sret(%struct.point767) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point767, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo768([4 x i32] %0) #0 {
  %2 = alloca %struct.point768, align 4
  %3 = alloca %struct.point768, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo769(ptr dead_on_unwind noalias writable sret(%struct.point769) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point769, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo770(ptr dead_on_unwind noalias writable sret(%struct.point770) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point770, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo771(ptr dead_on_unwind noalias writable sret(%struct.point771) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point771, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo772(ptr dead_on_unwind noalias writable sret(%struct.point772) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point772, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo773([4 x i32] %0) #0 {
  %2 = alloca %struct.point773, align 4
  %3 = alloca %struct.point773, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo774(ptr dead_on_unwind noalias writable sret(%struct.point774) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point774, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo775(ptr dead_on_unwind noalias writable sret(%struct.point775) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point775, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo776(ptr dead_on_unwind noalias writable sret(%struct.point776) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point776, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo777(ptr dead_on_unwind noalias writable sret(%struct.point777) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point777, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo778([3 x i32] %0) #0 {
  %2 = alloca %struct.point778, align 4
  %3 = alloca %struct.point778, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo779([4 x i32] %0) #0 {
  %2 = alloca %struct.point779, align 4
  %3 = alloca %struct.point779, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo780([4 x i32] %0) #0 {
  %2 = alloca %struct.point780, align 4
  %3 = alloca %struct.point780, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo781([4 x i32] %0) #0 {
  %2 = alloca %struct.point781, align 4
  %3 = alloca %struct.point781, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo782(ptr dead_on_unwind noalias writable sret(%struct.point782) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point782, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo783(ptr dead_on_unwind noalias writable sret(%struct.point783) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point783, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo784([4 x i32] %0) #0 {
  %2 = alloca %struct.point784, align 4
  %3 = alloca %struct.point784, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo785([4 x i32] %0) #0 {
  %2 = alloca %struct.point785, align 4
  %3 = alloca %struct.point785, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo786([4 x i32] %0) #0 {
  %2 = alloca %struct.point786, align 4
  %3 = alloca %struct.point786, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo787(ptr dead_on_unwind noalias writable sret(%struct.point787) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point787, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo788(ptr dead_on_unwind noalias writable sret(%struct.point788) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point788, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo789([4 x i32] %0) #0 {
  %2 = alloca %struct.point789, align 4
  %3 = alloca %struct.point789, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo790(ptr dead_on_unwind noalias writable sret(%struct.point790) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point790, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo791(ptr dead_on_unwind noalias writable sret(%struct.point791) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point791, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo792(ptr dead_on_unwind noalias writable sret(%struct.point792) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point792, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo793(ptr dead_on_unwind noalias writable sret(%struct.point793) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point793, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo794([4 x i32] %0) #0 {
  %2 = alloca %struct.point794, align 4
  %3 = alloca %struct.point794, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo795(ptr dead_on_unwind noalias writable sret(%struct.point795) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point795, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo796(ptr dead_on_unwind noalias writable sret(%struct.point796) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point796, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo797(ptr dead_on_unwind noalias writable sret(%struct.point797) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point797, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo798(ptr dead_on_unwind noalias writable sret(%struct.point798) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point798, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo799([2 x i32] %0) #0 {
  %2 = alloca %struct.point799, align 4
  %3 = alloca %struct.point799, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo800([2 x i32] %0) #0 {
  %2 = alloca %struct.point800, align 4
  %3 = alloca %struct.point800, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo801([2 x i32] %0) #0 {
  %2 = alloca %struct.point801, align 4
  %3 = alloca %struct.point801, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo802([3 x i32] %0) #0 {
  %2 = alloca %struct.point802, align 4
  %3 = alloca %struct.point802, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo803([3 x i32] %0) #0 {
  %2 = alloca %struct.point803, align 4
  %3 = alloca %struct.point803, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo804([3 x i32] %0) #0 {
  %2 = alloca %struct.point804, align 4
  %3 = alloca %struct.point804, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo805([2 x i32] %0) #0 {
  %2 = alloca %struct.point805, align 4
  %3 = alloca %struct.point805, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo806([3 x i32] %0) #0 {
  %2 = alloca %struct.point806, align 4
  %3 = alloca %struct.point806, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo807([3 x i32] %0) #0 {
  %2 = alloca %struct.point807, align 4
  %3 = alloca %struct.point807, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo808([3 x i32] %0) #0 {
  %2 = alloca %struct.point808, align 4
  %3 = alloca %struct.point808, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo809([3 x i32] %0) #0 {
  %2 = alloca %struct.point809, align 4
  %3 = alloca %struct.point809, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo810([3 x i32] %0) #0 {
  %2 = alloca %struct.point810, align 4
  %3 = alloca %struct.point810, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo811([4 x i32] %0) #0 {
  %2 = alloca %struct.point811, align 4
  %3 = alloca %struct.point811, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo812([4 x i32] %0) #0 {
  %2 = alloca %struct.point812, align 4
  %3 = alloca %struct.point812, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo813([4 x i32] %0) #0 {
  %2 = alloca %struct.point813, align 4
  %3 = alloca %struct.point813, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo814([4 x i32] %0) #0 {
  %2 = alloca %struct.point814, align 4
  %3 = alloca %struct.point814, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo815([3 x i32] %0) #0 {
  %2 = alloca %struct.point815, align 4
  %3 = alloca %struct.point815, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo816([4 x i32] %0) #0 {
  %2 = alloca %struct.point816, align 4
  %3 = alloca %struct.point816, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo817([4 x i32] %0) #0 {
  %2 = alloca %struct.point817, align 4
  %3 = alloca %struct.point817, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo818([4 x i32] %0) #0 {
  %2 = alloca %struct.point818, align 4
  %3 = alloca %struct.point818, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo819([4 x i32] %0) #0 {
  %2 = alloca %struct.point819, align 4
  %3 = alloca %struct.point819, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo820([2 x i32] %0) #0 {
  %2 = alloca %struct.point820, align 4
  %3 = alloca %struct.point820, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo821([3 x i32] %0) #0 {
  %2 = alloca %struct.point821, align 4
  %3 = alloca %struct.point821, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo822([3 x i32] %0) #0 {
  %2 = alloca %struct.point822, align 4
  %3 = alloca %struct.point822, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo823([3 x i32] %0) #0 {
  %2 = alloca %struct.point823, align 4
  %3 = alloca %struct.point823, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo824([4 x i32] %0) #0 {
  %2 = alloca %struct.point824, align 4
  %3 = alloca %struct.point824, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo825([4 x i32] %0) #0 {
  %2 = alloca %struct.point825, align 4
  %3 = alloca %struct.point825, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo826([3 x i32] %0) #0 {
  %2 = alloca %struct.point826, align 4
  %3 = alloca %struct.point826, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo827([3 x i32] %0) #0 {
  %2 = alloca %struct.point827, align 4
  %3 = alloca %struct.point827, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo828([3 x i32] %0) #0 {
  %2 = alloca %struct.point828, align 4
  %3 = alloca %struct.point828, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo829([4 x i32] %0) #0 {
  %2 = alloca %struct.point829, align 4
  %3 = alloca %struct.point829, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo830([4 x i32] %0) #0 {
  %2 = alloca %struct.point830, align 4
  %3 = alloca %struct.point830, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo831([3 x i32] %0) #0 {
  %2 = alloca %struct.point831, align 4
  %3 = alloca %struct.point831, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo832([4 x i32] %0) #0 {
  %2 = alloca %struct.point832, align 4
  %3 = alloca %struct.point832, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo833([4 x i32] %0) #0 {
  %2 = alloca %struct.point833, align 4
  %3 = alloca %struct.point833, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo834([4 x i32] %0) #0 {
  %2 = alloca %struct.point834, align 4
  %3 = alloca %struct.point834, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo835([4 x i32] %0) #0 {
  %2 = alloca %struct.point835, align 4
  %3 = alloca %struct.point835, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo836([3 x i32] %0) #0 {
  %2 = alloca %struct.point836, align 4
  %3 = alloca %struct.point836, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo837([4 x i32] %0) #0 {
  %2 = alloca %struct.point837, align 4
  %3 = alloca %struct.point837, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo838([4 x i32] %0) #0 {
  %2 = alloca %struct.point838, align 4
  %3 = alloca %struct.point838, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo839([4 x i32] %0) #0 {
  %2 = alloca %struct.point839, align 4
  %3 = alloca %struct.point839, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo840([4 x i32] %0) #0 {
  %2 = alloca %struct.point840, align 4
  %3 = alloca %struct.point840, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo841([3 x i32] %0) #0 {
  %2 = alloca %struct.point841, align 4
  %3 = alloca %struct.point841, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo842([4 x i32] %0) #0 {
  %2 = alloca %struct.point842, align 4
  %3 = alloca %struct.point842, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo843([4 x i32] %0) #0 {
  %2 = alloca %struct.point843, align 4
  %3 = alloca %struct.point843, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo844([4 x i32] %0) #0 {
  %2 = alloca %struct.point844, align 4
  %3 = alloca %struct.point844, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo845(ptr dead_on_unwind noalias writable sret(%struct.point845) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point845, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo846(ptr dead_on_unwind noalias writable sret(%struct.point846) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point846, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo847([4 x i32] %0) #0 {
  %2 = alloca %struct.point847, align 4
  %3 = alloca %struct.point847, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo848([4 x i32] %0) #0 {
  %2 = alloca %struct.point848, align 4
  %3 = alloca %struct.point848, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo849([4 x i32] %0) #0 {
  %2 = alloca %struct.point849, align 4
  %3 = alloca %struct.point849, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo850(ptr dead_on_unwind noalias writable sret(%struct.point850) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point850, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo851(ptr dead_on_unwind noalias writable sret(%struct.point851) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point851, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo852([4 x i32] %0) #0 {
  %2 = alloca %struct.point852, align 4
  %3 = alloca %struct.point852, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo853(ptr dead_on_unwind noalias writable sret(%struct.point853) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point853, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo854(ptr dead_on_unwind noalias writable sret(%struct.point854) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point854, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo855(ptr dead_on_unwind noalias writable sret(%struct.point855) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point855, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo856(ptr dead_on_unwind noalias writable sret(%struct.point856) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point856, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo857([4 x i32] %0) #0 {
  %2 = alloca %struct.point857, align 4
  %3 = alloca %struct.point857, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo858(ptr dead_on_unwind noalias writable sret(%struct.point858) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point858, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo859(ptr dead_on_unwind noalias writable sret(%struct.point859) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point859, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo860(ptr dead_on_unwind noalias writable sret(%struct.point860) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point860, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo861(ptr dead_on_unwind noalias writable sret(%struct.point861) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point861, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo862([3 x i32] %0) #0 {
  %2 = alloca %struct.point862, align 4
  %3 = alloca %struct.point862, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo863([4 x i32] %0) #0 {
  %2 = alloca %struct.point863, align 4
  %3 = alloca %struct.point863, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo864([4 x i32] %0) #0 {
  %2 = alloca %struct.point864, align 4
  %3 = alloca %struct.point864, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo865([4 x i32] %0) #0 {
  %2 = alloca %struct.point865, align 4
  %3 = alloca %struct.point865, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo866(ptr dead_on_unwind noalias writable sret(%struct.point866) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point866, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo867(ptr dead_on_unwind noalias writable sret(%struct.point867) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point867, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo868([4 x i32] %0) #0 {
  %2 = alloca %struct.point868, align 4
  %3 = alloca %struct.point868, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo869([4 x i32] %0) #0 {
  %2 = alloca %struct.point869, align 4
  %3 = alloca %struct.point869, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo870([4 x i32] %0) #0 {
  %2 = alloca %struct.point870, align 4
  %3 = alloca %struct.point870, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo871(ptr dead_on_unwind noalias writable sret(%struct.point871) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point871, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo872(ptr dead_on_unwind noalias writable sret(%struct.point872) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point872, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo873([4 x i32] %0) #0 {
  %2 = alloca %struct.point873, align 4
  %3 = alloca %struct.point873, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo874(ptr dead_on_unwind noalias writable sret(%struct.point874) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point874, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo875(ptr dead_on_unwind noalias writable sret(%struct.point875) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point875, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo876(ptr dead_on_unwind noalias writable sret(%struct.point876) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point876, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo877(ptr dead_on_unwind noalias writable sret(%struct.point877) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point877, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo878([4 x i32] %0) #0 {
  %2 = alloca %struct.point878, align 4
  %3 = alloca %struct.point878, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo879(ptr dead_on_unwind noalias writable sret(%struct.point879) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point879, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo880(ptr dead_on_unwind noalias writable sret(%struct.point880) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point880, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo881(ptr dead_on_unwind noalias writable sret(%struct.point881) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point881, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo882(ptr dead_on_unwind noalias writable sret(%struct.point882) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point882, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo883([2 x i32] %0) #0 {
  %2 = alloca %struct.point883, align 4
  %3 = alloca %struct.point883, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo884([2 x i32] %0) #0 {
  %2 = alloca %struct.point884, align 4
  %3 = alloca %struct.point884, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo885([3 x i32] %0) #0 {
  %2 = alloca %struct.point885, align 4
  %3 = alloca %struct.point885, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo886([3 x i32] %0) #0 {
  %2 = alloca %struct.point886, align 4
  %3 = alloca %struct.point886, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo887([3 x i32] %0) #0 {
  %2 = alloca %struct.point887, align 4
  %3 = alloca %struct.point887, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo888([3 x i32] %0) #0 {
  %2 = alloca %struct.point888, align 4
  %3 = alloca %struct.point888, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo889([3 x i32] %0) #0 {
  %2 = alloca %struct.point889, align 4
  %3 = alloca %struct.point889, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo890([3 x i32] %0) #0 {
  %2 = alloca %struct.point890, align 4
  %3 = alloca %struct.point890, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo891([3 x i32] %0) #0 {
  %2 = alloca %struct.point891, align 4
  %3 = alloca %struct.point891, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo892([4 x i32] %0) #0 {
  %2 = alloca %struct.point892, align 4
  %3 = alloca %struct.point892, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo893([4 x i32] %0) #0 {
  %2 = alloca %struct.point893, align 4
  %3 = alloca %struct.point893, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo894([3 x i32] %0) #0 {
  %2 = alloca %struct.point894, align 4
  %3 = alloca %struct.point894, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo895([4 x i32] %0) #0 {
  %2 = alloca %struct.point895, align 4
  %3 = alloca %struct.point895, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo896([4 x i32] %0) #0 {
  %2 = alloca %struct.point896, align 4
  %3 = alloca %struct.point896, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo897([4 x i32] %0) #0 {
  %2 = alloca %struct.point897, align 4
  %3 = alloca %struct.point897, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo898([4 x i32] %0) #0 {
  %2 = alloca %struct.point898, align 4
  %3 = alloca %struct.point898, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo899([3 x i32] %0) #0 {
  %2 = alloca %struct.point899, align 4
  %3 = alloca %struct.point899, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo900([4 x i32] %0) #0 {
  %2 = alloca %struct.point900, align 4
  %3 = alloca %struct.point900, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo901([4 x i32] %0) #0 {
  %2 = alloca %struct.point901, align 4
  %3 = alloca %struct.point901, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo902([4 x i32] %0) #0 {
  %2 = alloca %struct.point902, align 4
  %3 = alloca %struct.point902, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo903([4 x i32] %0) #0 {
  %2 = alloca %struct.point903, align 4
  %3 = alloca %struct.point903, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [2 x i32] @demo904([2 x i32] %0) #0 {
  %2 = alloca %struct.point904, align 4
  %3 = alloca %struct.point904, align 4
  store [2 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 8, i1 false)
  %4 = load [2 x i32], ptr %2, align 4
  ret [2 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo905([3 x i32] %0) #0 {
  %2 = alloca %struct.point905, align 4
  %3 = alloca %struct.point905, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo906([3 x i32] %0) #0 {
  %2 = alloca %struct.point906, align 4
  %3 = alloca %struct.point906, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo907([3 x i32] %0) #0 {
  %2 = alloca %struct.point907, align 4
  %3 = alloca %struct.point907, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo908([4 x i32] %0) #0 {
  %2 = alloca %struct.point908, align 4
  %3 = alloca %struct.point908, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo909([4 x i32] %0) #0 {
  %2 = alloca %struct.point909, align 4
  %3 = alloca %struct.point909, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo910([3 x i32] %0) #0 {
  %2 = alloca %struct.point910, align 4
  %3 = alloca %struct.point910, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo911([3 x i32] %0) #0 {
  %2 = alloca %struct.point911, align 4
  %3 = alloca %struct.point911, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo912([3 x i32] %0) #0 {
  %2 = alloca %struct.point912, align 4
  %3 = alloca %struct.point912, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo913([4 x i32] %0) #0 {
  %2 = alloca %struct.point913, align 4
  %3 = alloca %struct.point913, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo914([4 x i32] %0) #0 {
  %2 = alloca %struct.point914, align 4
  %3 = alloca %struct.point914, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo915([3 x i32] %0) #0 {
  %2 = alloca %struct.point915, align 4
  %3 = alloca %struct.point915, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo916([4 x i32] %0) #0 {
  %2 = alloca %struct.point916, align 4
  %3 = alloca %struct.point916, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo917([4 x i32] %0) #0 {
  %2 = alloca %struct.point917, align 4
  %3 = alloca %struct.point917, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo918([4 x i32] %0) #0 {
  %2 = alloca %struct.point918, align 4
  %3 = alloca %struct.point918, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo919([4 x i32] %0) #0 {
  %2 = alloca %struct.point919, align 4
  %3 = alloca %struct.point919, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo920([3 x i32] %0) #0 {
  %2 = alloca %struct.point920, align 4
  %3 = alloca %struct.point920, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo921([4 x i32] %0) #0 {
  %2 = alloca %struct.point921, align 4
  %3 = alloca %struct.point921, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo922([4 x i32] %0) #0 {
  %2 = alloca %struct.point922, align 4
  %3 = alloca %struct.point922, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo923([4 x i32] %0) #0 {
  %2 = alloca %struct.point923, align 4
  %3 = alloca %struct.point923, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo924([4 x i32] %0) #0 {
  %2 = alloca %struct.point924, align 4
  %3 = alloca %struct.point924, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo925([3 x i32] %0) #0 {
  %2 = alloca %struct.point925, align 4
  %3 = alloca %struct.point925, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo926([4 x i32] %0) #0 {
  %2 = alloca %struct.point926, align 4
  %3 = alloca %struct.point926, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo927([4 x i32] %0) #0 {
  %2 = alloca %struct.point927, align 4
  %3 = alloca %struct.point927, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo928([4 x i32] %0) #0 {
  %2 = alloca %struct.point928, align 4
  %3 = alloca %struct.point928, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo929(ptr dead_on_unwind noalias writable sret(%struct.point929) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point929, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo930(ptr dead_on_unwind noalias writable sret(%struct.point930) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point930, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo931([4 x i32] %0) #0 {
  %2 = alloca %struct.point931, align 4
  %3 = alloca %struct.point931, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo932([4 x i32] %0) #0 {
  %2 = alloca %struct.point932, align 4
  %3 = alloca %struct.point932, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo933([4 x i32] %0) #0 {
  %2 = alloca %struct.point933, align 4
  %3 = alloca %struct.point933, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo934(ptr dead_on_unwind noalias writable sret(%struct.point934) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point934, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo935(ptr dead_on_unwind noalias writable sret(%struct.point935) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point935, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo936([4 x i32] %0) #0 {
  %2 = alloca %struct.point936, align 4
  %3 = alloca %struct.point936, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo937(ptr dead_on_unwind noalias writable sret(%struct.point937) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point937, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo938(ptr dead_on_unwind noalias writable sret(%struct.point938) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point938, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo939(ptr dead_on_unwind noalias writable sret(%struct.point939) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point939, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo940(ptr dead_on_unwind noalias writable sret(%struct.point940) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point940, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo941([4 x i32] %0) #0 {
  %2 = alloca %struct.point941, align 4
  %3 = alloca %struct.point941, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo942(ptr dead_on_unwind noalias writable sret(%struct.point942) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point942, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo943(ptr dead_on_unwind noalias writable sret(%struct.point943) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point943, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo944(ptr dead_on_unwind noalias writable sret(%struct.point944) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point944, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo945(ptr dead_on_unwind noalias writable sret(%struct.point945) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point945, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo946([3 x i32] %0) #0 {
  %2 = alloca %struct.point946, align 4
  %3 = alloca %struct.point946, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo947([4 x i32] %0) #0 {
  %2 = alloca %struct.point947, align 4
  %3 = alloca %struct.point947, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo948([4 x i32] %0) #0 {
  %2 = alloca %struct.point948, align 4
  %3 = alloca %struct.point948, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo949([4 x i32] %0) #0 {
  %2 = alloca %struct.point949, align 4
  %3 = alloca %struct.point949, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo950(ptr dead_on_unwind noalias writable sret(%struct.point950) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point950, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo951(ptr dead_on_unwind noalias writable sret(%struct.point951) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point951, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo952([4 x i32] %0) #0 {
  %2 = alloca %struct.point952, align 4
  %3 = alloca %struct.point952, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo953([4 x i32] %0) #0 {
  %2 = alloca %struct.point953, align 4
  %3 = alloca %struct.point953, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo954([4 x i32] %0) #0 {
  %2 = alloca %struct.point954, align 4
  %3 = alloca %struct.point954, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo955(ptr dead_on_unwind noalias writable sret(%struct.point955) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point955, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo956(ptr dead_on_unwind noalias writable sret(%struct.point956) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point956, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo957([4 x i32] %0) #0 {
  %2 = alloca %struct.point957, align 4
  %3 = alloca %struct.point957, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo958(ptr dead_on_unwind noalias writable sret(%struct.point958) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point958, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo959(ptr dead_on_unwind noalias writable sret(%struct.point959) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point959, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo960(ptr dead_on_unwind noalias writable sret(%struct.point960) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point960, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo961(ptr dead_on_unwind noalias writable sret(%struct.point961) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point961, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo962([4 x i32] %0) #0 {
  %2 = alloca %struct.point962, align 4
  %3 = alloca %struct.point962, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo963(ptr dead_on_unwind noalias writable sret(%struct.point963) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point963, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo964(ptr dead_on_unwind noalias writable sret(%struct.point964) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point964, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo965(ptr dead_on_unwind noalias writable sret(%struct.point965) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point965, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo966(ptr dead_on_unwind noalias writable sret(%struct.point966) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point966, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo967([3 x i32] %0) #0 {
  %2 = alloca %struct.point967, align 4
  %3 = alloca %struct.point967, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo968([3 x i32] %0) #0 {
  %2 = alloca %struct.point968, align 4
  %3 = alloca %struct.point968, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo969([3 x i32] %0) #0 {
  %2 = alloca %struct.point969, align 4
  %3 = alloca %struct.point969, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo970([3 x i32] %0) #0 {
  %2 = alloca %struct.point970, align 4
  %3 = alloca %struct.point970, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo971([4 x i32] %0) #0 {
  %2 = alloca %struct.point971, align 4
  %3 = alloca %struct.point971, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo972([4 x i32] %0) #0 {
  %2 = alloca %struct.point972, align 4
  %3 = alloca %struct.point972, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo973([3 x i32] %0) #0 {
  %2 = alloca %struct.point973, align 4
  %3 = alloca %struct.point973, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo974([4 x i32] %0) #0 {
  %2 = alloca %struct.point974, align 4
  %3 = alloca %struct.point974, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo975([4 x i32] %0) #0 {
  %2 = alloca %struct.point975, align 4
  %3 = alloca %struct.point975, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo976([4 x i32] %0) #0 {
  %2 = alloca %struct.point976, align 4
  %3 = alloca %struct.point976, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo977([4 x i32] %0) #0 {
  %2 = alloca %struct.point977, align 4
  %3 = alloca %struct.point977, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo978([4 x i32] %0) #0 {
  %2 = alloca %struct.point978, align 4
  %3 = alloca %struct.point978, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo979(ptr dead_on_unwind noalias writable sret(%struct.point979) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point979, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo980(ptr dead_on_unwind noalias writable sret(%struct.point980) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point980, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo981(ptr dead_on_unwind noalias writable sret(%struct.point981) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point981, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo982(ptr dead_on_unwind noalias writable sret(%struct.point982) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point982, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo983([4 x i32] %0) #0 {
  %2 = alloca %struct.point983, align 4
  %3 = alloca %struct.point983, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo984(ptr dead_on_unwind noalias writable sret(%struct.point984) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point984, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo985(ptr dead_on_unwind noalias writable sret(%struct.point985) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point985, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo986(ptr dead_on_unwind noalias writable sret(%struct.point986) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point986, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo987(ptr dead_on_unwind noalias writable sret(%struct.point987) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point987, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo988([3 x i32] %0) #0 {
  %2 = alloca %struct.point988, align 4
  %3 = alloca %struct.point988, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo989([3 x i32] %0) #0 {
  %2 = alloca %struct.point989, align 4
  %3 = alloca %struct.point989, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo990([3 x i32] %0) #0 {
  %2 = alloca %struct.point990, align 4
  %3 = alloca %struct.point990, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo991([4 x i32] %0) #0 {
  %2 = alloca %struct.point991, align 4
  %3 = alloca %struct.point991, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo992([4 x i32] %0) #0 {
  %2 = alloca %struct.point992, align 4
  %3 = alloca %struct.point992, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo993([4 x i32] %0) #0 {
  %2 = alloca %struct.point993, align 4
  %3 = alloca %struct.point993, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo994([3 x i32] %0) #0 {
  %2 = alloca %struct.point994, align 4
  %3 = alloca %struct.point994, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo995([4 x i32] %0) #0 {
  %2 = alloca %struct.point995, align 4
  %3 = alloca %struct.point995, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo996([4 x i32] %0) #0 {
  %2 = alloca %struct.point996, align 4
  %3 = alloca %struct.point996, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo997([4 x i32] %0) #0 {
  %2 = alloca %struct.point997, align 4
  %3 = alloca %struct.point997, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo998([4 x i32] %0) #0 {
  %2 = alloca %struct.point998, align 4
  %3 = alloca %struct.point998, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo999([4 x i32] %0) #0 {
  %2 = alloca %struct.point999, align 4
  %3 = alloca %struct.point999, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1000(ptr dead_on_unwind noalias writable sret(%struct.point1000) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1000, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1001(ptr dead_on_unwind noalias writable sret(%struct.point1001) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1001, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1002(ptr dead_on_unwind noalias writable sret(%struct.point1002) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1002, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1003(ptr dead_on_unwind noalias writable sret(%struct.point1003) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1003, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1004([4 x i32] %0) #0 {
  %2 = alloca %struct.point1004, align 4
  %3 = alloca %struct.point1004, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1005(ptr dead_on_unwind noalias writable sret(%struct.point1005) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1005, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1006(ptr dead_on_unwind noalias writable sret(%struct.point1006) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1006, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1007(ptr dead_on_unwind noalias writable sret(%struct.point1007) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1007, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1008(ptr dead_on_unwind noalias writable sret(%struct.point1008) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1008, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo1009([3 x i32] %0) #0 {
  %2 = alloca %struct.point1009, align 4
  %3 = alloca %struct.point1009, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1010([4 x i32] %0) #0 {
  %2 = alloca %struct.point1010, align 4
  %3 = alloca %struct.point1010, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1011([4 x i32] %0) #0 {
  %2 = alloca %struct.point1011, align 4
  %3 = alloca %struct.point1011, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1012([4 x i32] %0) #0 {
  %2 = alloca %struct.point1012, align 4
  %3 = alloca %struct.point1012, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1013(ptr dead_on_unwind noalias writable sret(%struct.point1013) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1013, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1014(ptr dead_on_unwind noalias writable sret(%struct.point1014) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1014, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1015([4 x i32] %0) #0 {
  %2 = alloca %struct.point1015, align 4
  %3 = alloca %struct.point1015, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1016([4 x i32] %0) #0 {
  %2 = alloca %struct.point1016, align 4
  %3 = alloca %struct.point1016, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1017([4 x i32] %0) #0 {
  %2 = alloca %struct.point1017, align 4
  %3 = alloca %struct.point1017, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1018(ptr dead_on_unwind noalias writable sret(%struct.point1018) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1018, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1019(ptr dead_on_unwind noalias writable sret(%struct.point1019) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1019, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1020([4 x i32] %0) #0 {
  %2 = alloca %struct.point1020, align 4
  %3 = alloca %struct.point1020, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1021(ptr dead_on_unwind noalias writable sret(%struct.point1021) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1021, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1022(ptr dead_on_unwind noalias writable sret(%struct.point1022) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1022, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1023(ptr dead_on_unwind noalias writable sret(%struct.point1023) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1023, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1024(ptr dead_on_unwind noalias writable sret(%struct.point1024) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1024, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1025([4 x i32] %0) #0 {
  %2 = alloca %struct.point1025, align 4
  %3 = alloca %struct.point1025, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1026(ptr dead_on_unwind noalias writable sret(%struct.point1026) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1026, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1027(ptr dead_on_unwind noalias writable sret(%struct.point1027) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1027, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1028(ptr dead_on_unwind noalias writable sret(%struct.point1028) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1028, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1029(ptr dead_on_unwind noalias writable sret(%struct.point1029) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1029, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [3 x i32] @demo1030([3 x i32] %0) #0 {
  %2 = alloca %struct.point1030, align 4
  %3 = alloca %struct.point1030, align 4
  store [3 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 12, i1 false)
  %4 = load [3 x i32], ptr %2, align 4
  ret [3 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1031([4 x i32] %0) #0 {
  %2 = alloca %struct.point1031, align 4
  %3 = alloca %struct.point1031, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1032([4 x i32] %0) #0 {
  %2 = alloca %struct.point1032, align 4
  %3 = alloca %struct.point1032, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1033([4 x i32] %0) #0 {
  %2 = alloca %struct.point1033, align 4
  %3 = alloca %struct.point1033, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1034(ptr dead_on_unwind noalias writable sret(%struct.point1034) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1034, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1035(ptr dead_on_unwind noalias writable sret(%struct.point1035) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1035, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1036([4 x i32] %0) #0 {
  %2 = alloca %struct.point1036, align 4
  %3 = alloca %struct.point1036, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1037([4 x i32] %0) #0 {
  %2 = alloca %struct.point1037, align 4
  %3 = alloca %struct.point1037, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1038([4 x i32] %0) #0 {
  %2 = alloca %struct.point1038, align 4
  %3 = alloca %struct.point1038, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1039(ptr dead_on_unwind noalias writable sret(%struct.point1039) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1039, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1040(ptr dead_on_unwind noalias writable sret(%struct.point1040) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1040, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1041([4 x i32] %0) #0 {
  %2 = alloca %struct.point1041, align 4
  %3 = alloca %struct.point1041, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1042(ptr dead_on_unwind noalias writable sret(%struct.point1042) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1042, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1043(ptr dead_on_unwind noalias writable sret(%struct.point1043) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1043, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1044(ptr dead_on_unwind noalias writable sret(%struct.point1044) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1044, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1045(ptr dead_on_unwind noalias writable sret(%struct.point1045) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1045, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local [4 x i32] @demo1046([4 x i32] %0) #0 {
  %2 = alloca %struct.point1046, align 4
  %3 = alloca %struct.point1046, align 4
  store [4 x i32] %0, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %2, ptr align 4 %3, i32 16, i1 false)
  %4 = load [4 x i32], ptr %2, align 4
  ret [4 x i32] %4
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1047(ptr dead_on_unwind noalias writable sret(%struct.point1047) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1047, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1048(ptr dead_on_unwind noalias writable sret(%struct.point1048) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1048, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1049(ptr dead_on_unwind noalias writable sret(%struct.point1049) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1049, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

; Function Attrs: noinline nounwind optnone
define dso_local void @demo1050(ptr dead_on_unwind noalias writable sret(%struct.point1050) align 4 %0, [5 x i32] %1) #0 {
  %3 = alloca %struct.point1050, align 4
  store [5 x i32] %1, ptr %3, align 4
  call void @llvm.memcpy.p0.p0.i32(ptr align 4 %0, ptr align 4 %3, i32 20, i1 false)
  ret void
}

attributes #0 = { noinline nounwind optnone "frame-pointer"="all" "no-trapping-math"="true" "stack-protector-buffer-size"="8" "target-features"="+atomctl,+bool,+clamps,+coprocessor,+debug,+density,+dfpaccel,+div32,+exception,+fp,+highpriinterrupts,+interrupt,+loop,+mac16,+memctl,+minmax,+miscsr,+mul32,+mul32high,+nsa,+prid,+regprotect,+rvector,+s32c1i,+sext,+threadptr,+timerint,+windowed" }
attributes #1 = { nocallback nofree nounwind willreturn memory(argmem: readwrite) }

!llvm.module.flags = !{!0, !1}
!llvm.ident = !{!2}

!0 = !{i32 1, !"wchar_size", i32 4}
!1 = !{i32 7, !"frame-pointer", i32 2}
!2 = !{!"clang version 19.1.2 (https://github.com/espressif/llvm-project 510a078c1ad4aee4460818bcb38ff0ba3fbf6a83)"}
